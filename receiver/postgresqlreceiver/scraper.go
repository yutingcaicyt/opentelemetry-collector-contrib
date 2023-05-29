// Copyright The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package postgresqlreceiver // import "github.com/open-telemetry/opentelemetry-collector-contrib/receiver/postgresqlreceiver"

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"go.opentelemetry.io/collector/featuregate"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/receiver"
	"go.opentelemetry.io/collector/receiver/scrapererror"
	"go.uber.org/zap"

	"github.com/open-telemetry/opentelemetry-collector-contrib/receiver/postgresqlreceiver/internal/metadata"
)

var (
	emitMetricsWithoutResourceAttributesFeatureGate = featuregate.GlobalRegistry().MustRegister(
		"receiver.postgresql.emitMetricsWithoutResourceAttributes",
		featuregate.StageAlpha,
		featuregate.WithRegisterDescription("Postgresql metrics are transitioning from being reported with identifying metric attributes "+
			"to being identified via resource attributes in order to fit the OpenTelemetry specification. This feature "+
			"gate controls emitting the old metrics without resource attributes. For more details, see: "+
			"https://github.com/open-telemetry/opentelemetry-collector-contrib/blob/main/receiver/postgresqlreceiver/README.md#feature-gate-configurations"),
		featuregate.WithRegisterReferenceURL("https://github.com/open-telemetry/opentelemetry-collector-contrib/issues/12960"),
	)
	emitMetricsWithResourceAttributesFeatureGate = featuregate.GlobalRegistry().MustRegister(
		"receiver.postgresql.emitMetricsWithResourceAttributes",
		featuregate.StageBeta,
		featuregate.WithRegisterDescription("Postgresql metrics are transitioning from being reported with identifying metric attributes "+
			"to being identified via resource attributes in order to fit the OpenTelemetry specification. This feature "+
			"gate controls emitting the new metrics with resource attributes. For more details, see: "+
			"https://github.com/open-telemetry/opentelemetry-collector-contrib/blob/main/receiver/postgresqlreceiver/README.md#feature-gate-configurations"),
		featuregate.WithRegisterReferenceURL("https://github.com/open-telemetry/opentelemetry-collector-contrib/issues/12960"),
	)
)

type postgreSQLScraper struct {
	logger                               *zap.Logger
	config                               *Config
	clientFactory                        postgreSQLClientFactory
	mb                                   *metadata.MetricsBuilder
	emitMetricsWithoutResourceAttributes bool
	emitMetricsWithResourceAttributes    bool
}

type postgreSQLClientFactory interface {
	getClient(c *Config, database string) (client, error)
}

type defaultClientFactory struct{}

func (d *defaultClientFactory) getClient(c *Config, database string) (client, error) {
	return newPostgreSQLClient(postgreSQLConfig{
		username: c.Username,
		password: c.Password,
		database: database,
		tls:      c.TLSClientSetting,
		address:  c.NetAddr,
	})
}

func newPostgreSQLScraper(
	settings receiver.CreateSettings,
	config *Config,
	clientFactory postgreSQLClientFactory,
) *postgreSQLScraper {
	return &postgreSQLScraper{
		logger:                               settings.Logger,
		config:                               config,
		clientFactory:                        clientFactory,
		mb:                                   metadata.NewMetricsBuilder(config.MetricsBuilderConfig, settings),
		emitMetricsWithResourceAttributes:    emitMetricsWithResourceAttributesFeatureGate.IsEnabled(),
		emitMetricsWithoutResourceAttributes: emitMetricsWithoutResourceAttributesFeatureGate.IsEnabled(),
	}
}

type dbRetrieval struct {
	sync.RWMutex
	activityMap map[databaseName]int64
	dbSizeMap   map[databaseName]int64
	dbStats     map[databaseName]databaseStats
}

// scrape scrapes the metric stats, transforms them and attributes them into a metric slices.
func (p *postgreSQLScraper) scrape(ctx context.Context) (pmetric.Metrics, error) {
	databases := p.config.Databases
	listClient, err := p.clientFactory.getClient(p.config, "")
	if err != nil {
		p.logger.Error("Failed to initialize connection to postgres", zap.Error(err))
		return pmetric.NewMetrics(), err
	}
	defer listClient.Close()

	if len(databases) == 0 {
		dbList, err := listClient.listDatabases(ctx)
		if err != nil {
			p.logger.Error("Failed to request list of databases from postgres", zap.Error(err))
			return pmetric.NewMetrics(), err
		}
		databases = dbList
	}

	now := pcommon.NewTimestampFromTime(time.Now())

	var errs scrapererror.ScrapeErrors
	r := &dbRetrieval{
		activityMap: make(map[databaseName]int64),
		dbSizeMap:   make(map[databaseName]int64),
		dbStats:     make(map[databaseName]databaseStats),
	}
	p.retrieveDBMetrics(ctx, listClient, databases, r, &errs)

	for _, database := range databases {
		dbClient, err := p.clientFactory.getClient(p.config, database)
		if err != nil {
			errs.Add(err)
			p.logger.Error("Failed to initialize connection to postgres", zap.String("database", database), zap.Error(err))
			continue
		}
		defer dbClient.Close()
		numTables := p.collectTables(ctx, now, dbClient, database, &errs)

		p.recordDatabase(now, database, r, numTables)

		if p.emitMetricsWithResourceAttributes {
			p.collectIndexes(ctx, now, dbClient, database, &errs)
		}
	}

	if p.emitMetricsWithResourceAttributes {
		p.mb.RecordPostgresqlDatabaseCountDataPoint(now, int64(len(databases)))
		p.collectBGWriterStats(ctx, now, listClient, &errs)
		p.collectWalAge(ctx, now, listClient, &errs)
		p.collectReplicationStats(ctx, now, listClient, &errs)
		p.collectMaxConnections(ctx, now, listClient, &errs)
	}

	return p.mb.Emit(), errs.Combine()
}

func (p *postgreSQLScraper) retrieveDBMetrics(
	ctx context.Context,
	listClient client,
	databases []string,
	r *dbRetrieval,
	errs *scrapererror.ScrapeErrors,
) {
	wg := &sync.WaitGroup{}

	wg.Add(3)
	go p.retrieveBackends(ctx, wg, listClient, databases, r, errs)
	go p.retrieveDatabaseSize(ctx, wg, listClient, databases, r, errs)
	go p.retrieveDatabaseStats(ctx, wg, listClient, databases, r, errs)

	wg.Wait()
}

func (p *postgreSQLScraper) recordDatabase(now pcommon.Timestamp, db string, r *dbRetrieval, numTables int64) {
	dbName := databaseName(db)
	if p.emitMetricsWithResourceAttributes {
		p.mb.RecordPostgresqlTableCountDataPoint(now, numTables)
		if activeConnections, ok := r.activityMap[dbName]; ok {
			p.mb.RecordPostgresqlBackendsDataPointWithoutDatabase(now, activeConnections)
		}
		if size, ok := r.dbSizeMap[dbName]; ok {
			p.mb.RecordPostgresqlDbSizeDataPointWithoutDatabase(now, size)
		}
		if stats, ok := r.dbStats[dbName]; ok {
			p.mb.RecordPostgresqlCommitsDataPointWithoutDatabase(now, stats.transactionCommitted)
			p.mb.RecordPostgresqlRollbacksDataPointWithoutDatabase(now, stats.transactionRollback)
		}
		p.mb.EmitForResource(metadata.WithPostgresqlDatabaseName(db))
	} else {
		if activeConnections, ok := r.activityMap[dbName]; ok {
			p.mb.RecordPostgresqlBackendsDataPoint(now, activeConnections, db)
		}
		if size, ok := r.dbSizeMap[dbName]; ok {
			p.mb.RecordPostgresqlDbSizeDataPoint(now, size, db)
		}
		if stats, ok := r.dbStats[dbName]; ok {
			p.mb.RecordPostgresqlCommitsDataPoint(now, stats.transactionCommitted, db)
			p.mb.RecordPostgresqlRollbacksDataPoint(now, stats.transactionRollback, db)
		}
	}
}

func (p *postgreSQLScraper) collectTables(ctx context.Context, now pcommon.Timestamp, dbClient client, db string, errs *scrapererror.ScrapeErrors) (numTables int64) {
	blockReads, err := dbClient.getBlocksReadByTable(ctx, db)
	if err != nil {
		errs.AddPartial(1, err)
	}

	tableMetrics, err := dbClient.getDatabaseTableMetrics(ctx, db)
	if err != nil {
		errs.AddPartial(1, err)
	}

	for tableKey, tm := range tableMetrics {
		if p.emitMetricsWithResourceAttributes {
			p.mb.RecordPostgresqlRowsDataPointWithoutDatabaseAndTable(now, tm.dead, metadata.AttributeStateDead)
			p.mb.RecordPostgresqlRowsDataPointWithoutDatabaseAndTable(now, tm.live, metadata.AttributeStateLive)
			p.mb.RecordPostgresqlOperationsDataPointWithoutDatabaseAndTable(now, tm.inserts, metadata.AttributeOperationIns)
			p.mb.RecordPostgresqlOperationsDataPointWithoutDatabaseAndTable(now, tm.del, metadata.AttributeOperationDel)
			p.mb.RecordPostgresqlOperationsDataPointWithoutDatabaseAndTable(now, tm.upd, metadata.AttributeOperationUpd)
			p.mb.RecordPostgresqlOperationsDataPointWithoutDatabaseAndTable(now, tm.hotUpd, metadata.AttributeOperationHotUpd)
			p.mb.RecordPostgresqlTableSizeDataPoint(now, tm.size)
			p.mb.RecordPostgresqlTableVacuumCountDataPoint(now, tm.vacuumCount)

			br, ok := blockReads[tableKey]
			if ok {
				p.mb.RecordPostgresqlBlocksReadDataPointWithoutDatabaseAndTable(now, br.heapRead, metadata.AttributeSourceHeapRead)
				p.mb.RecordPostgresqlBlocksReadDataPointWithoutDatabaseAndTable(now, br.heapHit, metadata.AttributeSourceHeapHit)
				p.mb.RecordPostgresqlBlocksReadDataPointWithoutDatabaseAndTable(now, br.idxRead, metadata.AttributeSourceIdxRead)
				p.mb.RecordPostgresqlBlocksReadDataPointWithoutDatabaseAndTable(now, br.idxHit, metadata.AttributeSourceIdxHit)
				p.mb.RecordPostgresqlBlocksReadDataPointWithoutDatabaseAndTable(now, br.toastHit, metadata.AttributeSourceToastHit)
				p.mb.RecordPostgresqlBlocksReadDataPointWithoutDatabaseAndTable(now, br.toastRead, metadata.AttributeSourceToastHit)
				p.mb.RecordPostgresqlBlocksReadDataPointWithoutDatabaseAndTable(now, br.tidxRead, metadata.AttributeSourceTidxRead)
				p.mb.RecordPostgresqlBlocksReadDataPointWithoutDatabaseAndTable(now, br.tidxHit, metadata.AttributeSourceTidxHit)
			}
			p.mb.EmitForResource(
				metadata.WithPostgresqlDatabaseName(db),
				metadata.WithPostgresqlTableName(tm.table),
			)
		} else {
			p.mb.RecordPostgresqlRowsDataPoint(now, tm.dead, db, tm.table, metadata.AttributeStateDead)
			p.mb.RecordPostgresqlRowsDataPoint(now, tm.live, db, tm.table, metadata.AttributeStateLive)
			p.mb.RecordPostgresqlOperationsDataPoint(now, tm.inserts, db, tm.table, metadata.AttributeOperationIns)
			p.mb.RecordPostgresqlOperationsDataPoint(now, tm.del, db, tm.table, metadata.AttributeOperationDel)
			p.mb.RecordPostgresqlOperationsDataPoint(now, tm.upd, db, tm.table, metadata.AttributeOperationUpd)
			p.mb.RecordPostgresqlOperationsDataPoint(now, tm.hotUpd, db, tm.table, metadata.AttributeOperationHotUpd)

			br, ok := blockReads[tableKey]
			if ok {
				p.mb.RecordPostgresqlBlocksReadDataPoint(now, br.heapRead, db, br.table, metadata.AttributeSourceHeapRead)
				p.mb.RecordPostgresqlBlocksReadDataPoint(now, br.heapHit, db, br.table, metadata.AttributeSourceHeapHit)
				p.mb.RecordPostgresqlBlocksReadDataPoint(now, br.idxRead, db, br.table, metadata.AttributeSourceIdxRead)
				p.mb.RecordPostgresqlBlocksReadDataPoint(now, br.idxHit, db, br.table, metadata.AttributeSourceIdxHit)
				p.mb.RecordPostgresqlBlocksReadDataPoint(now, br.toastHit, db, br.table, metadata.AttributeSourceToastHit)
				p.mb.RecordPostgresqlBlocksReadDataPoint(now, br.toastRead, db, br.table, metadata.AttributeSourceToastRead)
				p.mb.RecordPostgresqlBlocksReadDataPoint(now, br.tidxRead, db, br.table, metadata.AttributeSourceTidxRead)
				p.mb.RecordPostgresqlBlocksReadDataPoint(now, br.tidxHit, db, br.table, metadata.AttributeSourceTidxHit)
			}
		}
	}
	return int64(len(tableMetrics))
}

func (p *postgreSQLScraper) collectIndexes(
	ctx context.Context,
	now pcommon.Timestamp,
	client client,
	database string,
	errs *scrapererror.ScrapeErrors,
) {
	idxStats, err := client.getIndexStats(ctx, database)
	if err != nil {
		errs.AddPartial(1, err)
		return
	}

	for _, stat := range idxStats {
		p.mb.RecordPostgresqlIndexScansDataPoint(now, stat.scans)
		p.mb.RecordPostgresqlIndexSizeDataPoint(now, stat.size)
		p.mb.EmitForResource(
			metadata.WithPostgresqlDatabaseName(stat.database),
			metadata.WithPostgresqlTableName(stat.table),
			metadata.WithPostgresqlIndexName(stat.index),
		)
	}
}

func (p *postgreSQLScraper) collectBGWriterStats(
	ctx context.Context,
	now pcommon.Timestamp,
	client client,
	errs *scrapererror.ScrapeErrors,
) {
	bgStats, err := client.getBGWriterStats(ctx)
	if err != nil {
		errs.AddPartial(1, err)
		return
	}

	p.mb.RecordPostgresqlBgwriterBuffersAllocatedDataPoint(now, bgStats.buffersAllocated)

	p.mb.RecordPostgresqlBgwriterBuffersWritesDataPoint(now, bgStats.bgWrites, metadata.AttributeBgBufferSourceBgwriter)
	p.mb.RecordPostgresqlBgwriterBuffersWritesDataPoint(now, bgStats.bufferBackendWrites, metadata.AttributeBgBufferSourceBackend)
	p.mb.RecordPostgresqlBgwriterBuffersWritesDataPoint(now, bgStats.bufferCheckpoints, metadata.AttributeBgBufferSourceCheckpoints)
	p.mb.RecordPostgresqlBgwriterBuffersWritesDataPoint(now, bgStats.bufferFsyncWrites, metadata.AttributeBgBufferSourceBackendFsync)

	p.mb.RecordPostgresqlBgwriterCheckpointCountDataPoint(now, bgStats.checkpointsReq, metadata.AttributeBgCheckpointTypeRequested)
	p.mb.RecordPostgresqlBgwriterCheckpointCountDataPoint(now, bgStats.checkpointsScheduled, metadata.AttributeBgCheckpointTypeScheduled)

	p.mb.RecordPostgresqlBgwriterDurationDataPoint(now, bgStats.checkpointSyncTime, metadata.AttributeBgDurationTypeSync)
	p.mb.RecordPostgresqlBgwriterDurationDataPoint(now, bgStats.checkpointWriteTime, metadata.AttributeBgDurationTypeWrite)

	p.mb.RecordPostgresqlBgwriterMaxwrittenDataPoint(now, bgStats.maxWritten)
}

func (p *postgreSQLScraper) collectMaxConnections(
	ctx context.Context,
	now pcommon.Timestamp,
	client client,
	errs *scrapererror.ScrapeErrors,
) {
	mc, err := client.getMaxConnections(ctx)
	if err != nil {
		errs.AddPartial(1, err)
		return
	}
	p.mb.RecordPostgresqlConnectionMaxDataPoint(now, mc)
}

func (p *postgreSQLScraper) collectReplicationStats(
	ctx context.Context,
	now pcommon.Timestamp,
	client client,
	errs *scrapererror.ScrapeErrors,
) {
	rss, err := client.getReplicationStats(ctx)
	if err != nil {
		errs.AddPartial(1, err)
		return
	}
	for _, rs := range rss {
		if rs.pendingBytes >= 0 {
			p.mb.RecordPostgresqlReplicationDataDelayDataPoint(now, rs.pendingBytes, rs.clientAddr)
		}
		if rs.writeLag >= 0 {
			p.mb.RecordPostgresqlWalLagDataPoint(now, rs.writeLag, metadata.AttributeWalOperationLagWrite, rs.clientAddr)
		}
		if rs.replayLag >= 0 {
			p.mb.RecordPostgresqlWalLagDataPoint(now, rs.replayLag, metadata.AttributeWalOperationLagReplay, rs.clientAddr)
		}
		if rs.flushLag >= 0 {
			p.mb.RecordPostgresqlWalLagDataPoint(now, rs.flushLag, metadata.AttributeWalOperationLagFlush, rs.clientAddr)
		}
	}
}

func (p *postgreSQLScraper) collectWalAge(
	ctx context.Context,
	now pcommon.Timestamp,
	client client,
	errs *scrapererror.ScrapeErrors,
) {
	walAge, err := client.getLatestWalAgeSeconds(ctx)
	if errors.Is(err, errNoLastArchive) {
		// return no error as there is no last archive to derive the value from
		return
	}
	if err != nil {
		errs.AddPartial(1, fmt.Errorf("unable to determine latest WAL age: %w", err))
		return
	}
	p.mb.RecordPostgresqlWalAgeDataPoint(now, walAge)
}

func (p *postgreSQLScraper) retrieveDatabaseStats(
	ctx context.Context,
	wg *sync.WaitGroup,
	client client,
	databases []string,
	r *dbRetrieval,
	errors *scrapererror.ScrapeErrors,
) {
	defer wg.Done()
	dbStats, err := client.getDatabaseStats(ctx, databases)
	if err != nil {
		p.logger.Error("Errors encountered while fetching commits and rollbacks", zap.Error(err))
		errors.AddPartial(1, err)
		return
	}
	r.Lock()
	r.dbStats = dbStats
	r.Unlock()
}

func (p *postgreSQLScraper) retrieveDatabaseSize(
	ctx context.Context,
	wg *sync.WaitGroup,
	client client,
	databases []string,
	r *dbRetrieval,
	errors *scrapererror.ScrapeErrors,
) {
	defer wg.Done()
	databaseSizeMetrics, err := client.getDatabaseSize(ctx, databases)
	if err != nil {
		p.logger.Error("Errors encountered while fetching database size", zap.Error(err))
		errors.AddPartial(1, err)
		return
	}
	r.Lock()
	r.dbSizeMap = databaseSizeMetrics
	r.Unlock()
}

func (p *postgreSQLScraper) retrieveBackends(
	ctx context.Context,
	wg *sync.WaitGroup,
	client client,
	databases []string,
	r *dbRetrieval,
	errors *scrapererror.ScrapeErrors,
) {
	defer wg.Done()
	activityByDB, err := client.getBackends(ctx, databases)
	if err != nil {
		errors.AddPartial(1, err)
		return
	}
	r.Lock()
	r.activityMap = activityByDB
	r.Unlock()
}
