package modo

import (
	"testing"
	"time"

	. "github.com/smartystreets/goconvey/convey"
)

func TestOutputter(t *testing.T) {
	Convey("Outputter", t, func() {
		Convey("Should successfully read from writer and send to callback", func() {
			dataToWrite := []byte("hello world")
			dataRead := []byte{}

			o := NewOutputter(func(d []byte) {
				dataRead = append(dataRead, d...)
			})

			w := o.GetWriter()
			for i := 0; i < len(dataToWrite); i++ {
				w.Write([]byte{dataToWrite[i]})
			}

			go o.Start()
			time.Sleep(50 * time.Millisecond)
			So(dataToWrite, ShouldResemble, dataRead)
		})
	})
}
