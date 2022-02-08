package ethstats

import (
	"time"
)

func (s *State) InitCleanCRON(days int) error {
	go func() error {
		every := time.Hour * 24
		ticker := time.NewTicker(every)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				err := s.DeleteOlderData(days)
				if err != nil {
					return err
				}
			}
		}
	}()

	return nil
}
