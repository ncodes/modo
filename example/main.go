package main

import (
	"fmt"

	"github.com/ncodes/modo"
)

func main() {
	m := modo.NewMoDo("8b86980c57704372fd1c0aedf3af5b35ad4d472d0df98110f87b9b3105d0ff2b", true, true, nil)
	m.Add(&modo.Do{
		Cmd: []string{"bash", "-c", `
			while :
			do
				echo "hello friend"
				sleep 3s
			done	
		`},
		OutputCB: func(b []byte, stdout bool) {
			fmt.Print(string(b))
		},
	})
	m.Do()
}
