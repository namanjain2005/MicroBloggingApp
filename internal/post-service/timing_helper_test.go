package postservice

import (
    "testing"
    "time"
)

func runTimed(t *testing.T, name string, fn func(t *testing.T)) {
    t.Run(name, func(t *testing.T) {
        start := time.Now()
        fn(t)
        elapsed := time.Since(start)
        t.Logf("duration: %.3fms", float64(elapsed.Nanoseconds())/1e6)
    })
}
