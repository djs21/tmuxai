package internal

import (
	"context"
	"fmt"

	"github.com/alvinunreal/tmuxai/config"
	"github.com/alvinunreal/tmuxai/logger"
	"github.com/chromedp/chromedp"
)

type BrowserClient struct {
	config    *config.Config
	allocCtx  context.Context
	cancel    context.CancelFunc
	browser   context.Context
	cdpCtx    context.Context
	connected bool
}

func NewBrowserClient(cfg *config.Config) *BrowserClient {
	return &BrowserClient{
		config: cfg,
	}
}

func (b *BrowserClient) Connect() error {
	if b.config.Browserless.Token == "" {
		return fmt.Errorf("browserless token required")
	}

	wsURL := b.config.Browserless.BaseURL + "?token=" + b.config.Browserless.Token

	// Create remote allocator
	allocCtx, allocCancel := chromedp.NewRemoteAllocator(context.Background(), wsURL)
	if allocCtx == nil {
		allocCancel()
		return fmt.Errorf("failed to create remote allocator")
	}

	b.allocCtx = allocCtx

	// Create browser context
	browserCtx, browserCancel := chromedp.NewContext(
		b.allocCtx,
		chromedp.WithDebugf(func(s string, i ...interface{}) { logger.Debug(s, i...) }),
		chromedp.WithLogf(func(s string, i ...interface{}) { logger.Info(s, i...) }),
	)
	b.browser = browserCtx

	// Create CDP context
	cdpCtx, cdpCancel := chromedp.NewContext(browserCtx,
		chromedp.WithDebugf(func(s string, i ...interface{}) { logger.Debug(s, i...) }),
		chromedp.WithLogf(func(s string, i ...interface{}) { logger.Info(s, i...) }),
	)
	b.cdpCtx = cdpCtx

	// Store the final cancel function that will cancel everything
	// We need to call all cancel functions in reverse order
	b.cancel = func() {
		cdpCancel()
		browserCancel()
		allocCancel()
	}

	b.connected = true

	logger.Info("Connected to Browserless at %s", wsURL)
	return nil
}

func (b *BrowserClient) Navigate(ctx context.Context, url string) error {
	if !b.connected {
		return fmt.Errorf("not connected")
	}
	err := chromedp.Navigate(url).Do(b.cdpCtx)
	if err != nil {
		logger.Error("Navigate failed: %v", err)
	}
	return err
}

func (b *BrowserClient) Screenshot(ctx context.Context) ([]byte, error) {
	if !b.connected {
		return nil, fmt.Errorf("not connected")
	}
	var buf []byte
	err := chromedp.Screenshot("body", &buf, chromedp.NodeVisible, chromedp.ByQuery).Do(b.cdpCtx)
	if err != nil {
		logger.Error("Screenshot failed: %v", err)
	}
	return buf, err
}

func (b *BrowserClient) GetText(ctx context.Context, selector string) (string, error) {
	if !b.connected {
		return "", fmt.Errorf("not connected")
	}
	var text string
	err := chromedp.Text(selector, &text, chromedp.ByQuery).Do(b.cdpCtx)
	if err != nil {
		logger.Error("GetText failed: %v", err)
	}
	return text, err
}

func (b *BrowserClient) Close() error {
	if b.cancel != nil {
		b.cancel()
		b.connected = false
	}
	return nil
}
