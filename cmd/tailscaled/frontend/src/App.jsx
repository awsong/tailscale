import { useState, useEffect, useRef } from "react";
import SidebarWithContentSeparator from "./components/sidebar";
import * as wails from "@wailsio/runtime";
import * as GreetService from "/bindings/main/GreetService.js";

function App() {
  const [isLoading, setIsLoading] = useState(true);
  const [error, setError] = useState(null);
  const [state, setState] = useState(null);
  const [netMap, setNetMap] = useState(null);
  const [prefs, setPrefs] = useState(null);
  const [ws, setWs] = useState(null);

  useEffect(() => {
    wails.Events.On("onMessage", (message) => {
      console.log("Unix Socket =================== :", message);
    });

    const ws = new WebSocket("ws://localhost:8000/echo");
    setWs(ws);
    ws.onopen = () => {
      // Optionally send a message to the server
      ws.send(JSON.stringify({ cmd: "init" }));
    };

    ws.onmessage = (event) => {
      GreetService.Send("Rcved a message from the WebSocket server");
      const notify = JSON.parse(
        event.data.replace(/:\s*(\d{16,}),/g, ':"$1",')
      );
      console.log("Received message:", notify);
      setIsLoading(false);
      if (notify.NetMap != null) {
        setNetMap(notify.NetMap);
      }
      if (notify.Prefs != null) {
        setPrefs(notify.Prefs);
      }
      if (notify.State != null) {
        setState(notify.State);
      }
      if (notify.BrowseToURL != null) {
        console.log(notify.BrowseToURL);
        if (navigator.userAgent.includes("wails.io")) {
          wails.Browser.OpenURL(notify.BrowseToURL);
        } else {
          window.open(notify.BrowseToURL, "_blank", "noreferrer");
        }
      }
    };

    ws.onerror = (error) => {
      console.error("WebSocket error:", error);
      setError(error);
      setIsLoading(false);
    };

    ws.onclose = function () {
      console.log("Disconnected from the server");
      // Automatically try to reconnect
      setTimeout(ws.connect, 1000);
    };
    // Clean up on unmount
    return () => {
      console.log("Clean up useEffect");
      ws.close();
    };
  }, []); // Empty dependency array means this effect runs once on mount

  if (isLoading) {
    return <div>Loading...</div>;
  }

  if (error) {
    return <div>Error: {error.message}</div>;
  }
  return (
    <div className="container flex flex-row">
      <SidebarWithContentSeparator
        state={state}
        prefs={prefs}
        netMap={netMap}
        ws={ws}
      />
      <div className="flex flex-row gap-4"></div>
    </div>
  );
}

export default App;
