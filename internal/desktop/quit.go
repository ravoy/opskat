package desktop

import "sync/atomic"

// QuitGate keeps native quit hooks nonblocking while the frontend acknowledges
// confirmation and saves its work. Native dispatch must remain available during
// those acknowledgements. A successful preparation retries the native quit once.
func QuitGate(prepare func() bool, retry func()) func() bool {
	var preparing, approved atomic.Bool
	return func() bool {
		if approved.Load() {
			return true
		}
		if preparing.CompareAndSwap(false, true) {
			go func() {
				defer preparing.Store(false)
				if prepare() {
					approved.Store(true)
					retry()
				}
			}()
		}
		return false
	}
}
