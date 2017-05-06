package modo

import (
	"testing"

	docker "github.com/fsouza/go-dockerclient"
	. "github.com/smartystreets/goconvey/convey"
)

func TestMoDo(t *testing.T) {

	client, err := docker.NewClient(DockerSock)
	if err != nil {
		t.Fatalf("failed to create docker client: %s", err)
	}

	deleteContainer := func(id string) {
		err := client.RemoveContainer(docker.RemoveContainerOptions{
			ID:    id,
			Force: true,
		})
		if err != nil {
			t.Fatalf("failed to start container: %s", err)
		}
		return
	}

	container, err := client.CreateContainer(docker.CreateContainerOptions{
		Name: "test",
		Config: &docker.Config{
			Image: "busybox",
			Cmd:   []string{"sleep", "1h"},
		},
	})
	if err != nil {
		t.Fatalf("failed to create container: %s", err)
	}

	err = client.StartContainer(container.ID, nil)
	if err != nil {
		deleteContainer(container.ID)
		t.Fatalf("failed to start container: %s", err)
	}

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

				modo := NewMoDo(container.ID, true, false, outputCb)
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

				modo := NewMoDo(container.ID, true, false, outputCb)
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

				modo := NewMoDo(container.ID, true, false, outputCb)
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

				modo := NewMoDo(container.ID, true, false, outputCb)
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

				modo := NewMoDo(container.ID, true, false, outputCb)
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

		Convey("Test State Callback", func() {

			// Convey("Should receive all lifecycle state calls", func() {
			// 	var receivedStates = []State{}
			// 	t1 := &Do{
			// 		Cmd:               []string{"sleep", "2"},
			// 		AbortSeriesOnFail: false,
			// 		KeepOutput:        true,
			// 		StateCB: func(state State, task *Do) {
			// 			receivedStates = append(receivedStates, state)
			// 		},
			// 	}
			// 	modo := NewMoDo(container.ID, true, false, nil)
			// 	modo.Add(t1)
			// 	errs, err := modo.Do()
			// 	So(len(errs), ShouldEqual, 0)
			// 	So(err, ShouldBeNil)
			// 	So(len(receivedStates), ShouldEqual, 3)
			// 	So(receivedStates, ShouldResemble, []State{Before, Executing, After})
			// })

			Convey("Should receive all lifecycle state calls in MoDo instance state callback", func() {
				var receivedStates = []State{}
				t1 := &Do{
					Cmd:               []string{"sleep", "2"},
					AbortSeriesOnFail: false,
					KeepOutput:        true,
				}
				modo := NewMoDo(container.ID, true, false, nil)
				modo.Add(t1)
				modo.SetStateCB(func(state State, task *Do) {
					receivedStates = append(receivedStates, state)
				})
				errs, err := modo.Do()
				So(len(errs), ShouldEqual, 0)
				So(err, ShouldBeNil)
				So(len(receivedStates), ShouldEqual, 3)
				So(receivedStates, ShouldResemble, []State{Before, Executing, After})
			})
		})

	})

	deleteContainer(container.ID)
}
