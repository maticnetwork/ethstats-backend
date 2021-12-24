package ethstats

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
)

func (s *Server) getLatestBlock(w http.ResponseWriter, r *http.Request) {
	block, err := s.state.GetLatestBlock()
	if err != nil {
		err := fmt.Sprint(err)
		w.Write([]byte(err))
		return
	}

	marshalledBlock, err := json.Marshal(block)
	if err != nil {
		err := fmt.Sprint(err)
		w.Write([]byte(err))
		return
	}

	w.WriteHeader(200)
	w.Write([]byte(marshalledBlock))

}

func (s *Server) getBlock(w http.ResponseWriter, r *http.Request) {

	var block *BlockDB
	query := r.URL.Query()
	filters, present := query["number"]

	if present {
		number, err := strconv.ParseInt(filters[0], 10, 64)
		if err != nil {
			err := fmt.Sprint(err)
			w.Write([]byte(err))
			return
		}
		block, err = s.state.GetBlockByNumber(int(number))
		if err != nil {
			err := fmt.Sprint(err)
			w.Write([]byte(err))
			return
		}
	} else {
		filters, present := query["hash"]
		if present {
			var err error
			block, err = s.state.GetBlockByHash(filters[0])
			if err != nil {
				err := fmt.Sprint(err)
				w.Write([]byte(err))
				return
			}
		}

	}

	marshalledBlock, err := json.Marshal(block)
	if err != nil {
		err := fmt.Sprint(err)
		w.Write([]byte(err))
		return
	}

	w.WriteHeader(200)
	w.Write([]byte(marshalledBlock))

}
