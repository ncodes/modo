// Package modo allows the ability to run commands in series or in parallel in a docker container
// while also allowing flexible behaviours like attaching a simple function to collect logs,
// stopping the series of commands if one fails or continuing and enabling privileged mode per command.
package modo

import (
	"fmt"
	"time"

	docker "github.com/ncodes/go-dockerclient"
)

// OutputFunc represents a callback function to receive command output
type OutputFunc func(d []byte, stdout bool)

// DockerSock points to docker socket file
var DockerSock = "unix:///var/run/docker.sock"

// Do defines a command to execute along with resources and directives specific to it.
// Setting `AbortSeriesOnFail` to true will force the execution of a series of Do task to be aborted
// if the command fails. Set `Privileged` to run this command in privileged mode.
// Attach an OutputCallback to receive stdout and stderr streams
type Do struct {
	Cmd               []string
	AbortSeriesOnFail bool
	Privileged        bool
	OutputCB          OutputFunc
	Output            []byte
	KeepOutput        bool
	ExitCode          int
	Done              bool
}

// MoDo defines a structure for collection
// commands to execute in series, manage and collect
// outputs and more
type MoDo struct {
	containerID string
	series      bool
	tasks       []*Do
	client      *docker.Client
	outputCB    OutputFunc
	privileged  bool
}

// NewMoDo creates and returns a new MoDo instance.
// Set series to true to run the commands in series. If series
// is false, all commands are executed in parallel.
// Set privileged to true if all commands should be run in privileged mode.
// Attach an output callback function to receive stdout/stderr streams
func NewMoDo(containerID string, series bool, privileged bool, outputCB OutputFunc) *MoDo {
	return &MoDo{
		containerID: containerID,
		series:      series,
		outputCB:    outputCB,
		privileged:  privileged,
	}
}

// Add adds a new command to execute.
func (m *MoDo) Add(do ...*Do) {
	m.tasks = append(m.tasks, do...)
}

// GetTasks the tasks
func (m *MoDo) GetTasks() []*Do {
	return m.tasks
}

func (m *MoDo) exec(task *Do) error {

	exec, err := m.client.CreateExec(docker.CreateExecOptions{
		Container:    m.containerID,
		Cmd:          task.Cmd,
		AttachStderr: !(m.outputCB == nil && task.OutputCB == nil),
		AttachStdout: !(m.outputCB == nil && task.OutputCB == nil),
		Privileged:   !(!m.privileged && !task.Privileged),
	})
	if err != nil {
		return err
	}

	outputFunc := task.OutputCB
	if outputFunc == nil {
		outputFunc = m.outputCB
	}

	privileged := task.Privileged
	if !privileged {
		privileged = m.privileged
	}

	outOutputter := NewOutputter(func(d []byte) {
		if outputFunc != nil {
			outputFunc(d, true)
		}
		if task.KeepOutput {
			task.Output = append(task.Output, d...)
		}
	})

	errOutputter := NewOutputter(func(d []byte) {
		if outputFunc != nil {
			outputFunc(d, false)
		}
		if task.KeepOutput {
			task.Output = append(task.Output, d...)
		}
	})

	go outOutputter.Start()
	go errOutputter.Start()

	err = m.client.StartExec(exec.ID, docker.StartExecOptions{
		OutputStream: outOutputter.GetWriter(),
		ErrorStream:  errOutputter.GetWriter(),
	})

	if err != nil {
		outOutputter.Stop()
		errOutputter.Stop()
		return err
	}

	// give the outputter some time read all the logs
	time.Sleep(50 * time.Millisecond)

	// stop the outputters
	errOutputter.Stop()
	outOutputter.Stop()

	execIns, err := m.client.InspectExec(exec.ID)
	if err != nil {
		return fmt.Errorf("failed to inspect exec: %s", err)
	}

	task.ExitCode = execIns.ExitCode

	return err
}

// Do execs all the commands in series or parallel.
// It returns a list of all task related error ([]error) and
// a general error.
func (m *MoDo) Do() ([]error, error) {

	var err error
	var errs []error

	m.client, err = docker.NewClient(DockerSock)
	if err != nil {
		return nil, err
	}

	// ensure container is running
	_, err = m.client.InspectContainer(m.containerID)
	if err != nil {
		return nil, err
	}

	for i, task := range m.tasks {
		if m.series {
			err = m.exec(task)
			task.Done = true
			if err != nil {
				return nil, err
			}

			if task.ExitCode != 0 {
				if task.AbortSeriesOnFail {
					errs = append(errs, fmt.Errorf("task: %d exited with exit code: %d", i, task.ExitCode))
					break
				}
				errs = append(errs, fmt.Errorf("task: %d exited with exit code: %d", i, task.ExitCode))
			}
		}
	}

	return errs, nil
}
