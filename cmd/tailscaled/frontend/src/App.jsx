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

  useEffect(() => {
    wails.Events.On("onMessage", (message) => {
      const notify = JSON.parse(
        message.data.replace(/:\s*(\d{16,}),/g, ':"$1",')
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
    });

    GreetService.Send(JSON.stringify({ cmd: "init" }));

    // Clean up on unmount
    return () => {
      console.log("Clean up useEffect");
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
      />
      <div className="flex flex-row gap-4"></div>
    </div>
  );
}

export default App;
