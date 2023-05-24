package dl

import (
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"

	alog "github.com/anacrolix/log"
	"github.com/anacrolix/torrent"
	"github.com/anacrolix/torrent/metainfo"
	"github.com/schollz/progressbar/v3"
	"go.elara.ws/logger/log"
)

var urlMatchRegex = regexp.MustCompile(`(magnet|torrent\+https?):.*`)

type TorrentDownloader struct{}

// Name always returns "file"
func (TorrentDownloader) Name() string {
	return "torrent"
}

// MatchURL returns true if the URL is a magnet link
// or an http(s) link with a "torrent+" prefix
func (TorrentDownloader) MatchURL(u string) bool {
	return urlMatchRegex.MatchString(u)
}

// Download downloads a file over the BitTorrent protocol.
func (TorrentDownloader) Download(opts Options) (Type, string, error) {
	cfg := torrent.NewDefaultClientConfig()
	cfg.DataDir = opts.Destination
	cfg.DisableWebseeds = true
	cfg.Logger.SetHandlers(alog.DiscardHandler)

	c, err := torrent.NewClient(cfg)
	if err != nil {
		return 0, "", err
	}
	defer c.Close()

	var t *torrent.Torrent
	if strings.HasPrefix(opts.URL, "magnet:") {
		t, err = c.AddMagnet(opts.URL)
		if err != nil {
			return 0, "", err
		}
		log.Info("Waiting for torrent metadata").Str("source", opts.Name).Send()
		<-t.GotInfo()
	} else if strings.HasPrefix(opts.URL, "torrent+") {
		log.Info("Downloading torrent file").Str("source", opts.Name).Send()

		res, err := http.Get(strings.TrimPrefix(opts.URL, "torrent+"))
		if err != nil {
			return 0, "", err
		}

		meta, err := metainfo.Load(res.Body)
		if err != nil {
			return 0, "", err
		}

		t, err = c.AddTorrent(meta)
		if err != nil {
			return 0, "", err
		}
	}

	t.DownloadAll()
	info := t.Info()
	info.BestName()

	var bar *progressbar.ProgressBar
	if opts.Progress != nil {
		bar = progressbar.NewOptions64(
			info.TotalLength(),
			progressbar.OptionSetDescription(info.Name),
			progressbar.OptionSetWriter(opts.Progress),
			progressbar.OptionShowBytes(true),
			progressbar.OptionSetWidth(10),
			progressbar.OptionThrottle(65*time.Millisecond),
			progressbar.OptionShowCount(),
			progressbar.OptionOnCompletion(func() {
				_, _ = io.WriteString(opts.Progress, "\n")
			}),
			progressbar.OptionSpinnerType(14),
			progressbar.OptionFullWidth(),
			progressbar.OptionSetRenderBlankState(true),
		)
		defer bar.Close()
	}

	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	for range ticker.C {
		info = t.Info()
		if t.Complete.Bool() {
			if info.IsDir() {
				return TypeDir, info.Name, nil
			} else {
				return TypeFile, info.Name, nil
			}
		}

		if bar != nil {
			bar.ChangeMax64(info.TotalLength())
			bar.Set64(t.BytesCompleted())
			stats := t.Stats()
			bar.Describe(fmt.Sprintf("%s [%d/%d]", info.Name, stats.ActivePeers, stats.TotalPeers))
		}
	}

	// This code should never execute because the loop will return from the function
	// once the torrent has finished downloading.
	panic("unreachable")
}
