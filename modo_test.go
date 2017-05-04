package modo

import (
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

func TestMoDo(t *testing.T) {
	Convey("MoDo", t, func() {

		Convey(".Add", func() {
			modo := NewMoDo("container_id", true, false, nil)
			t1 := &Do{Cmd: []string{"pwd"}}
			t2 := &Do{Cmd: []string{"ls"}}
			modo.Add(t1)
			modo.Add(t2)
			tasks := modo.GetTasks()
			So(tasks[0], ShouldResemble, t1)
			So(tasks[1], ShouldResemble, t2)
		})

		Convey(".Do", func() {

			Convey("Should return error if container does not exist", func() {
				modo := NewMoDo("container_id", true, false, nil)
				t1 := &Do{Cmd: []string{"pwd"}}
				t2 := &Do{Cmd: []string{"ls"}}
				modo.Add(t1)
				modo.Add(t2)
				_, err := modo.Do()
				So(err, ShouldNotBeNil)
				So(err.Error(), ShouldEqual, "No such container: container_id")
			})

			Convey("Should get the expected log output in callback", func() {

				var out []byte
				outputCb := func(d []byte, stdout bool) {
					out = append(out, d...)
				}

				modo := NewMoDo("56e65af437ba7510596f96d4a29e3e43e90c895ce7ff2910f2ec550ed0999f2f", true, false, outputCb)
				t1 := &Do{Cmd: []string{"echo", "hello world"}, AbortSeriesOnFail: false}
				modo.Add(t1)
				errs, err := modo.Do()
				So(err, ShouldBeNil)
				So(len(errs), ShouldEqual, 0)
				So(out, ShouldResemble, []byte("hello world\n"))
			})

			Convey("Should include invalid command error in returned tasks errors", func() {

				var out []byte
				outputCb := func(d []byte, stdout bool) {
					out = append(out, d...)
				}

				modo := NewMoDo("56e65af437ba7510596f96d4a29e3e43e90c895ce7ff2910f2ec550ed0999f2f", true, false, outputCb)
				t1 := &Do{Cmd: []string{"xyz", "hello world"}, AbortSeriesOnFail: false}
				modo.Add(t1)
				errs, err := modo.Do()
				So(err, ShouldBeNil)
				So(len(errs), ShouldEqual, 1)
			})

			Convey("Should stop processing command series if a task with AbortSeriesOnFail is true", func() {

				var out []byte

				outputCb := func(d []byte, stdout bool) {
					out = append(out, d...)
				}

				modo := NewMoDo("56e65af437ba7510596f96d4a29e3e43e90c895ce7ff2910f2ec550ed0999f2f", true, false, outputCb)
				t1 := &Do{Cmd: []string{"echo", "hello"}, AbortSeriesOnFail: false}
				t2 := &Do{Cmd: []string{"xyz", "hello world"}, AbortSeriesOnFail: true}
				t3 := &Do{Cmd: []string{"printf", "friend"}, AbortSeriesOnFail: false}
				modo.Add(t1, t2, t3)
				errs, err := modo.Do()
				So(err, ShouldBeNil)
				So(len(errs), ShouldEqual, 1)
				So(t1.Done, ShouldEqual, true)
				So(t1.ExitCode, ShouldEqual, 0)
				So(t2.Done, ShouldEqual, true)
				So(t2.ExitCode, ShouldNotEqual, 0)
				So(t3.Done, ShouldEqual, false)
			})

			Convey("Should continue processing other commands if all task's AbortSeriesOnFail are false", func() {

				var out []byte
				outputCb := func(d []byte, stdout bool) {
					out = append(out, d...)
				}

				modo := NewMoDo("56e65af437ba7510596f96d4a29e3e43e90c895ce7ff2910f2ec550ed0999f2f", true, false, outputCb)
				t1 := &Do{Cmd: []string{"echo", "hello"}, AbortSeriesOnFail: false}
				t2 := &Do{Cmd: []string{"xyz", "hello world"}, AbortSeriesOnFail: false}
				t3 := &Do{Cmd: []string{"printf", "friend"}, AbortSeriesOnFail: false}
				modo.Add(t1, t2, t3)
				errs, err := modo.Do()
				So(err, ShouldBeNil)
				So(len(errs), ShouldEqual, 1)
				So(t1.Done, ShouldEqual, true)
				So(t1.ExitCode, ShouldEqual, 0)
				So(t2.Done, ShouldEqual, true)
				So(t2.ExitCode, ShouldNotEqual, 0)
				So(t3.ExitCode, ShouldEqual, 0)
				So(t3.Done, ShouldEqual, true)
			})

			Convey("Should include tasks output in task.Output if KeepOutput is true", func() {

				var out []byte
				outputCb := func(d []byte, stdout bool) {
					out = append(out, d...)
				}

				modo := NewMoDo("56e65af437ba7510596f96d4a29e3e43e90c895ce7ff2910f2ec550ed0999f2f", true, false, outputCb)
				t1 := &Do{Cmd: []string{"printf", "hello"}, AbortSeriesOnFail: false, KeepOutput: true}
				t2 := &Do{Cmd: []string{"xyz", "hello world"}, AbortSeriesOnFail: false, KeepOutput: true}
				t3 := &Do{Cmd: []string{"printf", "friend"}, AbortSeriesOnFail: false, KeepOutput: true}
				modo.Add(t1, t2, t3)
				errs, err := modo.Do()
				So(err, ShouldBeNil)
				So(len(errs), ShouldEqual, 1)
				So(t1.Done, ShouldEqual, true)
				So(t1.Output, ShouldResemble, []byte("hello"))
				So(t1.ExitCode, ShouldEqual, 0)
				So(t2.Done, ShouldEqual, true)
				So(t2.ExitCode, ShouldNotEqual, 0)
				So(string(t2.Output), ShouldContainSubstring, `exec: \"xyz\": executable file not found`)
				So(t3.ExitCode, ShouldEqual, 0)
				So(t3.Done, ShouldEqual, true)
				So(t3.Output, ShouldResemble, []byte("friend"))
			})

		})

	})
}