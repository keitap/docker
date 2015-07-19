// +build windows

package windows

import (
	"errors"
	"fmt"

	"github.com/Sirupsen/logrus"
	"github.com/docker/docker/daemon/execdriver"
	"github.com/microsoft/hcsshim"
)

func (d *driver) Exec(c *execdriver.Command, processConfig *execdriver.ProcessConfig, pipes *execdriver.Pipes, startCallback execdriver.StartCallback) (int, error) {

	var (
		term     execdriver.Terminal
		err      error
		exitCode int32
	)

	active := d.activeContainers[c.ID]
	if active == nil {
		return -1, fmt.Errorf("Exec - No active container exists with ID %s", c.ID)
	}

	createProcessParms := hcsshim.CreateProcessParams{
		EmulateConsole:   processConfig.Tty, // Note NOT c.ProcessConfig.Tty
		WorkingDirectory: c.WorkingDir,
	}

	// Configure the environment for the process // Note NOT c.ProcessConfig.Tty
	createProcessParms.Environment = setupEnvironmentVariables(processConfig.Env)

	// While this should get caught earlier, just in case, validate that we
	// have something to run.
	if processConfig.Entrypoint == "" {
		err = errors.New("No entrypoint specified")
		logrus.Error(err)
		return -1, err
	}

	// Build the command line of the process
	createProcessParms.CommandLine = processConfig.Entrypoint
	for _, arg := range processConfig.Arguments {
		logrus.Debugln("appending ", arg)
		createProcessParms.CommandLine += " " + arg
	}
	logrus.Debugln("commandLine: ", createProcessParms.CommandLine)

	// Start the command running in the container.
	pid, stdin, stdout, stderr, err := hcsshim.CreateProcessInComputeSystem(c.ID, pipes.Stdin != nil, true, !processConfig.Tty, createProcessParms)
	if err != nil {
		logrus.Errorf("CreateProcessInComputeSystem() failed %s", err)
		return -1, err
	}

	// Now that the process has been launched, begin copying data to and from the named pipes for the std handles.
	if stdin != nil {
		stdinCopy(stdin, pipes.Stdin)
	}
	if stdout != nil {
		stdouterrCopy(stdout, "stdout", pipes.Stdout)
	}
	if stderr != nil {
		stdouterrCopy(stderr, "stderr", pipes.Stderr)
	}

	// Note NOT c.ProcessConfig.Tty
	if processConfig.Tty {
		term = NewTtyConsole(c.ID, pid)
	} else {
		term = NewStdConsole()
	}
	processConfig.Terminal = term

	// Invoke the start callback
	if startCallback != nil {
		startCallback(&c.ProcessConfig, int(pid))
	}

	if exitCode, err = hcsshim.WaitForProcessInComputeSystem(c.ID, pid); err != nil {
		logrus.Errorf("Failed to WaitForProcessInComputeSystem %s", err)
		return -1, err
	}

	// TODO Windows - Do something with this exit code
	logrus.Debugln("Exiting Run() with ExitCode 0", c.ID)
	return int(exitCode), nil
}
