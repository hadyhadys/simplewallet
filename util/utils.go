package util

import (
	"io"

	"github.com/inconshreveable/log15"
)

//WriteFlusher interface for flusher
type WriteFlusher interface {
	io.Writer
	Available() int // helps avoid splitting log records
	Flush() error
}

//StreamFlushHandler custom handler
func StreamFlushHandler(wf WriteFlusher, fmtr log15.Format, lvl log15.Lvl) log15.Handler {
	h := log15.FuncHandler(func(r *log15.Record) error {

		_, err := wf.Write(fmtr.Format(r))
		if err != nil {
			return err
		}
		return wf.Flush()
	})
	return log15.LvlFilterHandler(lvl, log15.SyncHandler(h))
}
