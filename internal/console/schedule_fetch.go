package console

import (
	"fmt"
	"log/slog"
	"sync"

	"github.com/kettari/location-bot/internal/bot"
	"github.com/kettari/location-bot/internal/config"
	"github.com/kettari/location-bot/internal/entity"
	"github.com/kettari/location-bot/internal/parser"
	"github.com/kettari/location-bot/internal/schedule"
	"github.com/kettari/location-bot/internal/scraper"
	"github.com/kettari/location-bot/internal/storage"
)

const (
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

	// Parsing pages
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

	// Parse pages
	prsr := parser.NewParser(parser.NewHtmlEngineV2())
	err = prsr.Parse(&result.Pages, sch)
	if err != nil {
		return err
	}

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

	if err = sch.SaveGames(); err != nil {
		return err
	}

	if err = sch.CheckAbsentGames(); err != nil {
		return err
	}

	slog.Info("schedule fetched successfully", "games_count", len(sch.Games))

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
	defer wg.Done()
	for job := range jobs {
		pageScraper := scraper.NewPage(job.url)
		err := pageScraper.LoadHtml()
		if err != nil {
			results <- Result{url: job.url, html: "", err: fmt.Errorf("job url %s (worker %d) failed to scrape page: %w", job.url, id, err)}
		} else {
			results <- Result{url: job.url, html: pageScraper.Html, err: nil}
		}
	}
}

func (cmd *ScheduleFetchCommand) collector(results <-chan Result, wg *sync.WaitGroup, pages *[]scraper.Page, errorFlag *bool) {
	defer wg.Done()
	for result := range results {
		if result.err == nil {
			*pages = append(*pages, scraper.Page{URL: result.url, Html: result.html})
		} else {
			slog.Warn("failed to fetch page", "url", result.url, "err", result.err)
			*errorFlag = true
		}
	}
}
