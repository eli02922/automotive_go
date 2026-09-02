package etl

import (
	"context"
	"time"

	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
)

type JobFunc func(ctx context.Context) error

type ScheduledJob struct {
	Name     string
	Interval time.Duration
	Run      JobFunc
}

// BackgroundWorker runs scheduled ETL jobs on fixed intervals using goroutines.
type BackgroundWorker struct {
	jobs []ScheduledJob
	log  *zap.Logger
}

func NewBackgroundWorker(log *zap.Logger) *BackgroundWorker {
	return &BackgroundWorker{log: log}
}

func (w *BackgroundWorker) Register(job ScheduledJob) {
	w.jobs = append(w.jobs, job)
}

// Start launches all registered jobs concurrently and blocks until ctx is cancelled.
func (w *BackgroundWorker) Start(ctx context.Context) error {
	g, ctx := errgroup.WithContext(ctx)

	for _, job := range w.jobs {
		job := job
		g.Go(func() error {
			return w.runJob(ctx, job)
		})
	}

	return g.Wait()
}

func (w *BackgroundWorker) runJob(ctx context.Context, job ScheduledJob) error {
	w.log.Info("background job started", zap.String("job", job.Name), zap.Duration("interval", job.Interval))
	ticker := time.NewTicker(job.Interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			w.log.Info("background job stopped", zap.String("job", job.Name))
			return nil
		case <-ticker.C:
			start := time.Now()
			if err := job.Run(ctx); err != nil {
				w.log.Error("background job failed",
					zap.String("job", job.Name),
					zap.Error(err),
					zap.Duration("elapsed", time.Since(start)),
				)
			} else {
				w.log.Info("background job completed",
					zap.String("job", job.Name),
					zap.Duration("elapsed", time.Since(start)),
				)
			}
		}
	}
}
