package main

import (
  "os"
  "io"
  "syscall"
  "fmt"
  "time"
  "strings"
  "strconv"
  "flag"
)

// A simple routine to grab the 1 minute avg load
// It will be sent through the c every second
func Load(c chan float64) {
  tick := time.Tick(time.Second)
  f, err := os.Open("/proc/loadavg")
  defer f.Close()
  if err != nil {
    fmt.Fprintln(os.Stderr, err)
    close(c)
    return
  }
  buff := make([]byte, 1024)
  var n int
  var load float64
  for _ = range tick {
    // Make sure we are at the beginning of the file
    if _, err = f.Seek(0, 0); err != nil {
      fmt.Fprintln(os.Stderr, err)
      close(c)
      return
    }
    if n, err = f.Read(buff); err != io.EOF && err != nil {
      fmt.Fprintln(os.Stderr, err)
      close(c)
      return
    }
    fields := strings.Fields(string(buff[:n]))
    if load, err = strconv.ParseFloat(fields[0], 64); err != nil {
      fmt.Fprintln(os.Stderr, err)
      close(c)
      return
    } else {
      c <- load
    }
  }
}

var (
  pid int
  help bool
  suspended bool
  ps *os.Process
  threshold float64
  sigcont os.Signal = syscall.SIGCONT
  sigstop os.Signal = syscall.SIGSTOP
)

func main() {
  var perr error
  flag.Parse()
  threshold, perr = strconv.ParseFloat(flag.Arg(0), 64)
  fmt.Println(flag.Arg(1))
  pid, perr = strconv.Atoi(flag.Arg(1))
  if perr != nil {
    fmt.Println("Suspends process when load reaches threshold")
    fmt.Println("pwatch load pid")
    return
  }
  if pid != -1 {
    var err error
    ps, err = os.FindProcess(pid)
    if err != nil {
      fmt.Fprintf(os.Stderr, "Could not find process with pid %d\n", pid)
      fmt.Fprintln(os.Stderr, err)
      return
    }
  }
  c := make(chan float64, 1)
  go Load(c)

  for load := range c {
    if load > threshold {
      if !suspended {
        fmt.Println("Going down")
        // signal stop
        if err := ps.Signal(sigstop); err != nil {
          fmt.Fprintf(os.Stderr, "Error stopping process: %s", err)
          // Should we exit?
          return
        }
      }
      suspended = true
    } else if suspended {
      fmt.Println("Bringing up")
      // signal continue
      if err := ps.Signal(sigcont); err != nil {
        fmt.Fprintf(os.Stderr, "Error starting process: %s", err)
        // should we exit?
        return
      }
      suspended = false
    }
  }
}
