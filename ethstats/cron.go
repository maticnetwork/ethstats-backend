package ethstats

import (
	"fmt"
	"time"
)

func (s *State) InitCleanCRON(days int) error {
	go func() {
		every := time.Hour * 24
		ticker := time.NewTicker(every)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				err := s.DeleteOlderData(days)
				if err != nil {
					fmt.Printf("[ERROR]: %v", err)
				}
			}
		}
	}()

	return nil
}
