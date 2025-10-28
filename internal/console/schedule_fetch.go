package console

import (
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/kettari/location-bot/internal/bot"
	"github.com/kettari/location-bot/internal/config"
	"github.com/kettari/location-bot/internal/entity"
	"github.com/kettari/location-bot/internal/parser"
	"github.com/kettari/location-bot/internal/schedule"
	"github.com/kettari/location-bot/internal/scraper"
	"github.com/kettari/location-bot/internal/storage"
)

const (
	rootURL      = "https://rolecon.ru"
	eventsURL    = "https://rolecon.ru/event/json-calendar?start=%s&end=%s"
	twoWeeks     = 24 * time.Hour * 14
	workersCount = 5
)

type ScheduleFetchCommand struct {
}

type Job struct {
	url string
}

type Result struct {
	url  string
	html string
	err  error
}

func NewScheduleFetchCommand() *ScheduleFetchCommand {
	cmd := ScheduleFetchCommand{}
	return &cmd
}

func (cmd *ScheduleFetchCommand) Name() string {
	return "schedule:fetch"
}

func (cmd *ScheduleFetchCommand) Description() string {
	return "fetches events from the Rolecon server and parses them to the database"
}

func (cmd *ScheduleFetchCommand) Run() error {
	slog.Info("fetching schedule")
	conf := config.GetConfig()

	// Use fetcher service to orchestrate CSRF and events collection
	fetcher := scraper.NewFetcher()
	result, err := fetcher.FetchAll(func(urls []string) ([]scraper.Page, error) {
		var pages []scraper.Page
		errorFlag := false
		cmd.dispatcher(urls, workersCount, &pages, &errorFlag)
		if errorFlag {
			slog.Warn("error flag was set, discarding results")
			return nil, fmt.Errorf("failed to fetch some pages")
		}
		return pages, nil
	})
	if err != nil {
		return err
	}

	// Parsing pages with both parsers for comparison
	slog.Debug("parsing pages")
	var sch *schedule.Schedule
	if !conf.DryRun {
		manager := storage.NewManager(conf.DbConnectionString)
		if err = manager.Connect(); err != nil {
			return err
		}
		sch = schedule.NewSchedule(manager)
	} else {
		slog.Info("DRY RUN MODE: skipping database connection")
		sch = schedule.NewSchedule(nil)
	}

	// Parse with original parser
	prsr := parser.NewParser(parser.NewHtmlEngine())
	err = prsr.Parse(&result.Pages, sch)
	if err != nil {
		return err
	}
	slog.Debug("pages parsed to games", "games_count", len(sch.Games))

	// Parse with V2 parser for comparison
	slog.Info("comparing V2 parser results")
	var schV2 []entity.Game
	prsrV2 := parser.NewParser(parser.NewHtmlEngineV2())
	for _, page := range result.Pages {
		games, err := prsrV2.ParseSinglePage(&page)
		if err != nil {
			slog.Warn("V2 parser error", "url", page.URL, "err", err)
			continue
		}
		schV2 = append(schV2, *games...)
	}

	// Compare results
	compareResults(sch.Games, schV2)

	// Register observers
	// Create bot with dependency injection (token and recipients)
	b, err := bot.CreateBot(conf.BotToken, conf.NotificationChatID)
	if err != nil {
		slog.Error("unable to create bot processor object", "error", err)
		return err
	}
	for k := range sch.Games {
		sch.Games[k].Register(entity.NewGameObserver(b))
		sch.Games[k].Register(entity.BecomeJoinableGameObserver(b))
		sch.Games[k].Register(entity.CancelledGameObserver(b))
	}

	slog.Debug("saving games", "games_count", len(sch.Games))
	if err = sch.SaveGames(); err != nil {
		return err
	}
	slog.Debug("games saved", "games_count", len(sch.Games))

	slog.Debug("check absent games")
	if err = sch.CheckAbsentGames(); err != nil {
		return err
	}
	slog.Debug("absent games checked")

	slog.Info("schedule fetched")

	return nil
}

// see [https://rksurwase.medium.com/efficient-concurrency-in-go-a-deep-dive-into-the-worker-pool-pattern-for-batch-processing-73cac5a5bdca]
func (cmd *ScheduleFetchCommand) dispatcher(urls []string, workerCount int, pages *[]scraper.Page, errorFlag *bool) {
	jobs := make(chan Job, len(urls))
	results := make(chan Result, len(urls))

	var wg sync.WaitGroup

	// Start workers
	wg.Add(workerCount)
	for w := 1; w <= workerCount; w++ {
		go cmd.worker(w, jobs, results, &wg)
	}

	// Start collecting results
	var resultsWg sync.WaitGroup
	resultsWg.Add(1)
	go cmd.collector(results, &resultsWg, pages, errorFlag)

	// Distribute jobs and wait for completion
	for _, url := range urls {
		jobs <- Job{url: url}
	}
	close(jobs)
	wg.Wait()
	close(results)

	// Ensure all results are collected
	resultsWg.Wait()
}

func (cmd *ScheduleFetchCommand) worker(id int, jobs <-chan Job, results chan<- Result, wg *sync.WaitGroup) {
	slog.Debug("scraping worker started", "id", id)
	defer wg.Done()
	for job := range jobs {
		slog.Debug("worker started processing url", "worker_id", id, "url", job.url)
		pageScraper := scraper.NewPage(job.url)
		err := pageScraper.LoadHtml()
		if err != nil {
			results <- Result{url: job.url, html: "", err: fmt.Errorf("job url %s (worker %d) failed to scrape page: %w", job.url, id, err)}
		} else {
			results <- Result{url: job.url, html: pageScraper.Html, err: nil}
		}
		slog.Debug("worker finished processing url", "worker_id", id, "url", job.url)
	}
	slog.Debug("scraping worker finished", "id", id)
}

func (cmd *ScheduleFetchCommand) collector(results <-chan Result, wg *sync.WaitGroup, pages *[]scraper.Page, errorFlag *bool) {
	slog.Debug("collecting results started")
	defer wg.Done()
	for result := range results {
		if result.err == nil {
			slog.Debug("collected event page", "url", result.url, "page_size", len(result.html))
			*pages = append(*pages, scraper.Page{URL: result.url, Html: result.html})
		} else {
			slog.Error("collected error", "url", result.url, "err", result.err)
			*errorFlag = true
		}
	}
	slog.Debug("collecting results finished")
}

func compareResults(original []entity.Game, v2 []entity.Game) {
	slog.Info("===========================================")
	slog.Info("PARSER COMPARISON RESULTS")
	slog.Info("===========================================")

	originalCount := len(original)
	v2Count := len(v2)

	slog.Info("Game counts",
		"original", originalCount,
		"v2", v2Count,
		"difference", originalCount-v2Count)

	if originalCount == 0 && v2Count == 0 {
		slog.Info("Both parsers found no games")
		return
	}

	// Count differences by field
	missingTitlesV2 := 0
	missingDescriptions := 0
	missingNotes := 0
	differentSeats := 0
	differentDates := 0

	for i, game := range original {
		if i < len(v2) {
			v2Game := v2[i]

			// Compare fields
			if v2Game.Title == "" && game.Title != "" {
				missingTitlesV2++
				slog.Warn("V2 missing title", "original", game.Title, "id", game.ExternalID)
			}
			if v2Game.Description == "" && game.Description != "" {
				missingDescriptions++
			}
			if game.Description != "" && v2Game.Description == "" {
				slog.Warn("V2 missing description", "original_len", len(game.Description), "id", game.ExternalID)
			}
			if v2Game.Notes == "" && game.Notes != "" {
				missingNotes++
			}
			if v2Game.SeatsFree != game.SeatsFree || v2Game.SeatsTotal != game.SeatsTotal {
				differentSeats++
				slog.Warn("Different seats",
					"id", game.ExternalID,
					"original", fmt.Sprintf("%d/%d", game.SeatsFree, game.SeatsTotal),
					"v2", fmt.Sprintf("%d/%d", v2Game.SeatsFree, v2Game.SeatsTotal))
			}
			if !game.Date.Equal(v2Game.Date) && !game.Date.IsZero() && !v2Game.Date.IsZero() {
				differentDates++
				slog.Warn("Different dates",
					"id", game.ExternalID,
					"original", game.Date.String(),
					"v2", v2Game.Date.String())
			}
		}
	}

	// Check for extra fields extracted by V2
	hasDescription := 0
	hasNotes := 0
	for _, game := range v2 {
		if game.Description != "" {
			hasDescription++
		}
		if game.Notes != "" {
			hasNotes++
		}
	}

	slog.Info("Comparison Summary",
		"missing_titles_v2", missingTitlesV2,
		"missing_descriptions", missingDescriptions,
		"missing_notes", missingNotes,
		"different_seats", differentSeats,
		"different_dates", differentDates,
		"v2_descriptions_extracted", hasDescription,
		"v2_notes_extracted", hasNotes)

	// Sample comparison of first 3 games
	sampleCount := 3
	if originalCount < sampleCount {
		sampleCount = originalCount
	}

	slog.Info(fmt.Sprintf("Sample of first %d games:", sampleCount))
	for i := 0; i < sampleCount && i < len(original); i++ {
		orig := original[i]
		slog.Info("Original parser game",
			"id", orig.ExternalID,
			"title", orig.Title[:min(50, len(orig.Title))],
			"description_len", len(orig.Description),
			"notes_len", len(orig.Notes),
			"seats", fmt.Sprintf("%d/%d", orig.SeatsFree, orig.SeatsTotal))
	}

	if len(v2) > 0 {
		slog.Info(fmt.Sprintf("Sample of first %d V2 games:", min(sampleCount, len(v2))))
		for i := 0; i < sampleCount && i < len(v2); i++ {
			v2Game := v2[i]
			slog.Info("V2 parser game",
				"id", v2Game.ExternalID,
				"title", v2Game.Title[:min(50, len(v2Game.Title))],
				"description_len", len(v2Game.Description),
				"notes_len", len(v2Game.Notes),
				"seats", fmt.Sprintf("%d/%d", v2Game.SeatsFree, v2Game.SeatsTotal))
		}
	}

	slog.Info("===========================================")
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
