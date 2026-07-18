package llmtrace

import (
	"context"
	"fmt"
	"time"
)

// RunSync pulls traces since last sync, analyzes them, and optionally writes scores.
func RunSync(ctx context.Context, client *Client, store *StateStore, opts SyncOptions) (*SyncReport, error) {
	start := time.Now()
	report := &SyncReport{
		FindingsBySeverity: make(map[string]int),
	}

	batchSize := opts.BatchSize
	if batchSize <= 0 {
		batchSize = 50
	}

	scorePrefix := opts.ScorePrefix
	if scorePrefix == "" {
		scorePrefix = "vg_"
	}

	// Determine start time
	var fromTime *time.Time
	if opts.From != nil {
		fromTime = opts.From
	} else if !store.State.LastSyncTime.IsZero() {
		fromTime = &store.State.LastSyncTime
	}

	analyzer := NewAnalyzer(scorePrefix)

	// First pass: build cost baseline from first batch
	baselinePage, err := client.FetchTraces(ctx, 1, batchSize, fromTime)
	if err != nil {
		return nil, fmt.Errorf("fetch baseline traces: %w", err)
	}
	analyzer.UpdateCostBaseline(baselinePage.Data)

	// Paginate through all traces
	page := 1
	for {
		select {
		case <-ctx.Done():
			report.Errors = append(report.Errors, "sync cancelled")
			report.DurationMs = time.Since(start).Milliseconds()
			return report, ctx.Err()
		default:
		}

		var tracesResp *TracesResponse
		if page == 1 {
			tracesResp = baselinePage // reuse first fetch
		} else {
			tracesResp, err = client.FetchTraces(ctx, page, batchSize, fromTime)
			if err != nil {
				report.Errors = append(report.Errors, fmt.Sprintf("page %d: %s", page, err))
				break
			}
		}

		if len(tracesResp.Data) == 0 {
			break
		}

		// Update cost baseline with each batch
		if page > 1 {
			analyzer.UpdateCostBaseline(tracesResp.Data)
		}

		for i := range tracesResp.Data {
			trace := &tracesResp.Data[i]

			// Fetch observations if not embedded
			if len(trace.Observations) == 0 {
				obs, obsErr := client.FetchObservations(ctx, trace.ID)
				if obsErr != nil {
					report.Errors = append(report.Errors, fmt.Sprintf("observations for %s: %s", trace.ID, obsErr))
				} else {
					trace.Observations = obs
				}
			}

			// Analyze
			analysis := analyzer.AnalyzeTrace(trace)
			report.TracesAnalyzed++

			for _, f := range analysis.Findings {
				report.FindingsTotal++
				report.FindingsBySeverity[f.Severity]++

				// Write scores back to provider
				if opts.WriteScores && !opts.DryRun {
					scoreReq := ScoreRequest{
						TraceID:       trace.ID,
						Name:          f.ScoreName,
						Value:         f.Score,
						DataType:      "NUMERIC",
						Comment:       fmt.Sprintf("[VectraGuard] %s: %s", f.Type, f.Detail),
						ObservationID: f.ObsID,
					}
					if postErr := client.PostScore(ctx, scoreReq); postErr != nil {
						report.Errors = append(report.Errors, fmt.Sprintf("post score %s: %s", trace.ID, postErr))
					} else {
						report.ScoresWritten++
					}
				}
			}
		}

		// Check if there are more pages
		if page >= tracesResp.Meta.TotalPages || tracesResp.Meta.TotalPages == 0 {
			break
		}
		page++
	}

	// Update sync state
	store.State.TracesTotal += report.TracesAnalyzed
	store.State.ScoresTotal += report.ScoresWritten
	if !opts.DryRun {
		if saveErr := store.Save(); saveErr != nil {
			report.Errors = append(report.Errors, fmt.Sprintf("save state: %s", saveErr))
		}
	}

	report.DurationMs = time.Since(start).Milliseconds()
	return report, nil
}
