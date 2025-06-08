package console

import (
	"fmt"
	"github.com/kettari/location-bot/internal/bot"
	"github.com/kettari/location-bot/internal/config"
	"github.com/kettari/location-bot/internal/entity"
	"github.com/kettari/location-bot/internal/parser"
	"github.com/kettari/location-bot/internal/schedule"
	"github.com/kettari/location-bot/internal/scraper"
	"github.com/kettari/location-bot/internal/storage"
	"log/slog"
	"sync"
	"time"
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

	// Get the root page
	slog.Debug("requesting page", "url", rootURL)
	page := scraper.NewPage(rootURL)
	if err := page.LoadHtml(); err != nil {
		return err
	}
	slog.Debug("initial page loaded", "size", len(page.Html), "cookies_count", len(page.Cookies))

	// Extract Csrf token and cookie
	csrf := scraper.NewCsrf(page)
	var err error
	if err = csrf.ExtractCsrfToken(); err != nil {
		return err
	}
	slog.Debug("found CSRF token", "token", csrf.Token)
	if err = csrf.ExtractCsrfCookie(); err != nil {
		return err
	}
	slog.Debug("found CSRF cookie", "cookie", csrf.Cookie)

	// Load events JSON
	url := fmt.Sprintf(eventsURL, time.Now().Format("2006-01-02"), time.Now().Add(twoWeeks).Format("2006-01-02"))
	slog.Debug("requesting events", "url", url)
	events := scraper.NewEvents(url, csrf)
	if err = events.LoadEvents(); err != nil {
		return err
	}
	slog.Debug("events page loaded", "size", len(events.JSON))
	slog.Debug("unmarshalling events")
	if err = events.UnmarshalEvents(); err != nil {
		return err
	}
	slog.Debug("events unmarshalled", "events_count", len(events.Events))

	// Load individual events pages
	slog.Debug("requesting events pages")
	var urls []string
	for _, event := range events.Events {
		urls = append(urls, rootURL+event.URL)
	}
	var pages []scraper.Page
	errorFlag := false
	cmd.dispatcher(urls, workersCount, &pages, &errorFlag)
	slog.Debug("collected events pages", "pages_count", len(pages))
	if errorFlag {
		slog.Warn("error flag was set, discarding results")
		return nil
	}

	// Parsing pages
	slog.Debug("parsing pages")
	manager := storage.NewManager(conf.DbConnectionString)
	if err = manager.Connect(); err != nil {
		return err
	}
	sch := schedule.NewSchedule(manager)
	prsr := parser.NewParser(parser.NewHtmlEngine())
	err = prsr.Parse(&pages, sch)
	if err != nil {
		return err
	}
	slog.Debug("pages parsed to games", "games_count", len(sch.Games))

	// Register observers
	b, err := bot.CreateBot(conf.NotificationChatID)
	if err != nil {
		slog.Error("unable to create bot processor object", "error", err)
		return err
	}
	for k, _ := range sch.Games {
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
		var err error
		err = pageScraper.LoadHtml()
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
