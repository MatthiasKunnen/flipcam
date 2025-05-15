package flipcamlib

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"os/exec"
	"path"
	"strings"
	"sync"
	"time"
)

type RtmpToHlsMuxer struct {

	// The URL to start listening on for incoming RTMP streams.
	Url string

	// The Prefix is prepended to every filename written by the muxer.
	Prefix string

	// The path where the playlist file should be written.
	PlaylistPath string

	cmd     *exec.Cmd
	mu      sync.Mutex
	stdin   io.WriteCloser
	stopped chan struct{}

	done    chan struct{}
	doneErr error
}

// Start starts the RTMP to HLS muxing process by listening on the specified URL.
// Waiting for the muxing to end can be done using Wait.
//
// url is the URL that the muxer should start listening on, it should start with rtmp://.
//
// playlistPath is the path where the playlist file will be written to.
// Segments will be placed next to the playlist file.
func (m *RtmpToHlsMuxer) Start() error {
	if !strings.HasPrefix(m.Url, "rtmp://") {
		return fmt.Errorf("url must begin with rtmp://")
	}
	if !strings.HasSuffix(m.PlaylistPath, ".m3u8") {
		return fmt.Errorf("playlist path must end with .m3u8")
	}

	m.mu.Lock()
	defer m.mu.Unlock()
	destinationDir := path.Dir(m.PlaylistPath)
	cmd := exec.Command("ffmpeg",
		"-loglevel", "warning",
		"-listen", "1", // Wait for connection
		"-i", m.Url,
		"-rtmp_live", "live",
		"-rtmp_buffer", "1000",
		"-c:v", "copy",
		"-an", // Drop audio, wasted bytes underwater
		"-f", "hls",
		"-hls_list_size", "0",
		"-hls_segment_type", "fmp4",
		"-hls_time", "1",
		"-hls_flags", "program_date_time+split_by_time",
		"-hls_playlist_type", "event",
		"-hls_segment_filename", path.Join(destinationDir, m.Prefix+"%d.mp4"),
		"-hls_fmp4_init_filename", m.Prefix+"init.mp4",
		m.PlaylistPath)
	log.Printf("[muxer]: cmd: %s\n", cmd)

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return fmt.Errorf("muxer: could not get stdin pipe: %w", err)
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("muxer: could not get stdout pipe: %w", err)
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("muxer: could not get stderr pipe: %w", err)
	}

	go func() {
		s := bufio.NewScanner(stdout)
		for s.Scan() {
			log.Printf("[muxer]: ffmpeg stdout: %s\n", s.Text())
		}
		if err := s.Err(); err != nil {
			log.Printf("[muxer]: error processing stdout: %v\n", err)
		}
	}()
	go func() {
		s := bufio.NewScanner(stderr)
		for s.Scan() {
			log.Printf("[muxer]: ffmpeg stderr: %s\n", s.Text())
		}
		if err := s.Err(); err != nil {
			log.Printf("[muxer]: error processing stderr: %v\n", err)
		}
	}()

	m.cmd = cmd
	m.stdin = stdin
	m.stopped = make(chan struct{})
	m.done = make(chan struct{})
	m.doneErr = nil

	err = cmd.Start()
	if err != nil {
		return fmt.Errorf("muxer: failed to start: %w", err)
	}
	log.Printf(
		"[muxer]: Ready for RTMP ingest at %s. Playlist will be at %s.\n",
		m.Url,
		m.PlaylistPath,
	)

	go func() {
		err := cmd.Wait()
		close(m.stopped)

		var exitError *exec.ExitError
		switch {
		case errors.As(err, &exitError) && exitError.ExitCode() == 255:
			// Assuming 255 is only used when exit signal is used, unsure
		default:
			m.doneErr = err
		}
		m.done <- struct{}{}
	}()

	return nil
}

// Wait waits for the muxing to end.
// Wait must be called in the same goroutine as Start.
func (m *RtmpToHlsMuxer) Wait() error {
	<-m.done
	return m.doneErr
}

// Shutdown stops the muxing process.
// This makes the Start function return after muxing stops.
// Restarting is possible by calling Start.
// Shutdown can be called when the muxer is not running.
// Shutdown is goroutine safe.
func (m *RtmpToHlsMuxer) Shutdown(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	defer func() {
		m.cmd = nil
	}()
	// Writing q to quit will close ffmpeg only if it is actively processing a stream.
	// If it was still waiting for a stream to ingest, this will not do anything.
	_, err := m.stdin.Write([]byte("q"))
	if err == nil {
		graceTime := 2 * time.Second
		if deadline, ok := ctx.Deadline(); ok && time.Until(deadline) < graceTime {
			// If deadline is set and closer than the default grace time, use half of the
			// time to deadline
			graceTime = time.Until(deadline) / 2
		}

		select {
		case <-m.stopped:
			return nil
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(graceTime):
			// Time to forcefully kill
		}
	} else {
		log.Printf("[muxer]: Failed to send quit signal to ffmpeg stdin: %v.\n", err)
	}

	log.Println("[muxer]: ffmpeg did not close after sending 'q', killing it.")
	err = m.cmd.Process.Kill()
	if err != nil {
		return fmt.Errorf("[muxer]: error when sending SIGKILL to ffmpeg: %w\n", err)
	}

	select {
	case <-m.stopped:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}
