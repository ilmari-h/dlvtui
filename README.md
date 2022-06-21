# dlv-tui

<img src="preview.gif" width="500">

***dlv-tui*** is a terminal interface for the [delve debugger](https://github.com/go-delve/delve). It's target audience is Go developers who prefer to use terminal only tools in their workflow. This frontend's goal is to provide all the functionality of the delve cli-debugger wrapped in an efficient tui.

## Usage

The client supports debugging by running an excecutable or by attaching to an existing process.
The debug target is the first argument, after which the following options can be provided:

- `-attach` - If enabled, attach debugger to process. Interpret first argument as PID.
- `-port` - The port dlv rpc server will listen to. (default "8181")

## Configuration

Keybindings, colors and behavior of the client are customizable via a yaml configuration file located at `$XDG_CONFIG_HOME/dlvtui/config.yaml`.
