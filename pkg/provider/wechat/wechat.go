package wechat

import (
	"encoding/csv"
	"fmt"
	"io"
	"log"
	"strings"

	"github.com/deb-sig/double-entry-generator/pkg/io/reader"
	"github.com/deb-sig/double-entry-generator/pkg/ir"
	"github.com/spf13/viper"
)

// Wechat is the provider for Wechat.
type Wechat struct {
	Statistics Statistics `json:"statistics,omitempty"`
	LineNum    int        `json:"line_num,omitempty"`
	Orders     []Order    `json:"orders,omitempty"`
	Owner      string     `json:"owner,omitempty"`
}

// New creates a new wechat provider.
func New() *Wechat {
	return &Wechat{
		Statistics: Statistics{},
		LineNum:    0,
		Orders:     make([]Order, 0),
	}
}

// Translate translates the alipay bill records to IR.
func (w *Wechat) Translate(filename string) (*ir.IR, error) {
	log.SetPrefix("[Provider-Wechat] ")

	billReader, err := reader.GetReader(filename)
	if err != nil {
		return nil, fmt.Errorf("can't get bill reader, err: %v", err)
	}

	csvReader := csv.NewReader(billReader)
	csvReader.LazyQuotes = true
	// If FieldsPerRecord is negative, no check is made and records
	// may have a variable number of fields.
	csvReader.FieldsPerRecord = -1

	for {
		line, err := csvReader.Read()

		if w.LineNum == 1 && strings.Contains(line[0], "微信昵称：") {
			owners := viper.GetStringMapString("wechat.owners")
			for owner, name := range owners {
				if strings.Contains(line[0], name) {
					w.Owner = owner
					log.Printf("Wechat bill with owner %s", w.Owner)
				}
			}
			if w.Owner == "" {
				return nil, fmt.Errorf("canot get bill owner with config %v", owners)
			}
		}

		if err == io.EOF {
			break
		} else if err != nil {
			return nil, err
		}

		w.LineNum++
		if w.LineNum <= 17 {
			// The first 17 lines are useless for us.
			continue
		}

		err = w.translateToOrders(line)
		if err != nil {
			return nil, fmt.Errorf("failed to translate bill: line %d: %v",
				w.LineNum, err)
		}
	}
	log.Printf("Finished to parse the file %s", filename)
	return w.convertToIR(), nil
}
