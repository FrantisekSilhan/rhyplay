package ffmpeg

import (
	"bufio"
	"net"
	"strconv"
	"strings"
)

type Progress struct {
	Percent float64
	Frames  int
	FPS     float64
	Done    bool
}

func StartProgressServer() (int, chan Progress, error) {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return 0, nil, err
	}

	progressChan := make(chan Progress, 10)
	port := l.Addr().(*net.TCPAddr).Port

	go func() {
		defer l.Close()
		defer close(progressChan)

		conn, err := l.Accept()
		if err != nil {
			return
		}
		defer conn.Close()

		scanner := bufio.NewScanner(conn)
		for scanner.Scan() {
			line := scanner.Text()
			parts := strings.SplitN(line, "=", 2)
			if len(parts) != 2 {
				continue
			}

			key, val := parts[0], parts[1]
			if key == "out_time_ms" {
				ms, _ := strconv.ParseInt(val, 10, 64)
				progressChan <- Progress{
					Percent: float64(ms) / 1000.0,
				}
			} else if key == "progress" && val == "end" {
				progressChan <- Progress{Done: true}
			}
		}
		if err := scanner.Err(); err != nil {
			return
		}
	}()

	return port, progressChan, nil
}
