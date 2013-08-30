package main

import (
	"github.com/dotcloud/docker/term"
	"github.com/garyburd/go-websocket/websocket"
	"github.com/kr/pty"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"syscall"
)

type wsWriter struct {
	stream chan []byte
}

func (w *wsWriter) Write(p []byte) (int, error) {
	w.stream <- p
	return len(p), nil
}

func fork(c chan []byte) chan []byte {
	f := make(chan []byte)
	go func() {
		for m := range c {
			f <- m
		}
		close(f)
	}()
	return f
}

func server(c chan []byte) {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		b, err := ioutil.ReadFile("index.html")
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Write(b)
	})

	http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {

		ws, err := websocket.Upgrade(w, r.Header, nil, 512, 512)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		defer ws.Close()

		f := fork(c)
		go func() {
			for msg := range f {
				if err := ws.WriteMessage(websocket.OpText, msg); err != nil {
					panic(err)
				}
			}
			ws.WriteMessage(websocket.OpClose, []byte{})
		}()

		for {
			_, _, err := ws.NextReader()
			if err != nil {
				return
			}
		}
	})

	if err := http.ListenAndServe(":8001", nil); err != nil {
		panic(err)
	}
}

func main() {

	cmd := exec.Command("zsh")
	m, s, err := pty.Open()
	if err != nil {
		panic(err)
	}
	state, _ := term.SetRawTerminal(os.Stdin.Fd())
	defer term.RestoreTerminal(os.Stdin.Fd(), state)

	size, _ := term.GetWinsize(os.Stdin.Fd())
	term.SetRawTerminal(m.Fd())
	term.SetWinsize(m.Fd(), size)

	c := make(chan []byte)
	ws := &wsWriter{c}
	go server(c)

	wout := io.MultiWriter(ws, os.Stdout)
	werr := io.MultiWriter(ws, os.Stderr)

	go io.Copy(wout, m)
	go io.Copy(werr, m)
	go io.Copy(m, os.Stdin)

	cmd.Stderr = s
	cmd.Stdout = s
	cmd.Stdin = s
	cmd.SysProcAttr = &syscall.SysProcAttr{Setctty: true, Setsid: true}

	if err := cmd.Run(); err != nil {
		panic(err)
	}
	close(c)
}
