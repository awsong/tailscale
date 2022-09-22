import { Terminal, ITerminalOptions } from "xterm"
import { FitAddon } from "xterm-addon-fit"
import { WebLinksAddon } from "xterm-addon-web-links"

export type SSHSessionDef = {
  username: string
  hostname: string
  /** Defaults to 5 seconds */
  timeoutSeconds?: number
}

export function runSSHSession(
  termContainerNode: HTMLDivElement,
  def: SSHSessionDef,
  ipn: IPN,
  onDone: () => void,
  onError?: (err: string) => void,
  terminalOptions?: ITerminalOptions
) {
  const parentWindow = termContainerNode.ownerDocument.defaultView ?? window
  const term = new Terminal({
    cursorBlink: true,
    allowProposedApi: true,
    ...terminalOptions,
  })

  const fitAddon = new FitAddon()
  term.loadAddon(fitAddon)
  term.open(termContainerNode)
  fitAddon.fit()

  const webLinksAddon = new WebLinksAddon((event, uri) =>
    event.view?.open(uri, "_blank", "noopener")
  )
  term.loadAddon(webLinksAddon)

  let onDataHook: ((data: string) => void) | undefined
  term.onData((e) => {
    onDataHook?.(e)
  })

  term.focus()

  let resizeObserver: ResizeObserver | undefined
  let handleBeforeUnload: ((e: BeforeUnloadEvent) => void) | undefined

  const sshSession = ipn.ssh(def.hostname, def.username, {
    writeFn(input) {
      term.write(input)
    },
    writeErrorFn(err) {
      onError?.(err)
      term.write(err)
    },
    setReadFn(hook) {
      onDataHook = hook
    },
    rows: term.rows,
    cols: term.cols,
    onDone() {
      resizeObserver?.disconnect()
      term.dispose()
      if (handleBeforeUnload) {
        parentWindow.removeEventListener("beforeunload", handleBeforeUnload)
      }
      onDone()
    },
    timeoutSeconds: def.timeoutSeconds,
  })

  // Make terminal and SSH session track the size of the containing DOM node.
  resizeObserver = new parentWindow.ResizeObserver(() => fitAddon.fit())
  resizeObserver.observe(termContainerNode)
  term.onResize(({ rows, cols }) => sshSession.resize(rows, cols))

  // Close the session if the user closes the window without an explicit
  // exit.
  handleBeforeUnload = () => sshSession.close()
  parentWindow.addEventListener("beforeunload", handleBeforeUnload)
}
