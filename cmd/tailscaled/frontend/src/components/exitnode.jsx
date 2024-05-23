import {
  Radio,
  Card,
  List,
  ListItem,
  ListItemPrefix,
  Typography,
} from "@material-tailwind/react";
import * as GreetService from "/bindings/main/GreetService.js";

function Icon() {
  return (
    <svg
      xmlns="http://www.w3.org/2000/svg"
      viewBox="0 0 24 24"
      fill="currentColor"
      className="h-full w-full scale-105"
    >
      <path
        fillRule="evenodd"
        d="M2.25 12c0-5.385 4.365-9.75 9.75-9.75s9.75 4.365 9.75 9.75-4.365 9.75-9.75 9.75S2.25 17.385 2.25 12zm13.36-1.814a.75.75 0 10-1.22-.872l-3.236 4.53L9.53 12.22a.75.75 0 00-1.06 1.06l2.25 2.25a.75.75 0 001.14-.094l3.75-5.25z"
        clipRule="evenodd"
      />
    </svg>
  );
}
function ExitNode({ netMap, prefs }) {
  let exits = netMap.Peers.filter((peer) => {
    if (peer.AllowedIPs.includes("0.0.0.0/0")) {
      return peer;
    }
  });

  return (
    <div className="flex flex-col pr-4">
      <Radio
        name="type"
        icon={<Icon />}
        className="w-4 h-4 border-gray-900/10 bg-gray-900/5 transition-all hover:before:opacity-0"
        checked={prefs.ExitNodeID === ""}
        onChange={() => {
          GreetService.Send(JSON.stringify({ cmd: "exitNode", data: "" }));
        }}
        label={
          <Typography color="blue-gray" className="text-sm">
            禁用出口节点
          </Typography>
        }
      />
      <hr className="my-2 border-blue-gray-50" />
      {exits.map((exit, index) => (
        <Radio
          name="type"
          icon={<Icon />}
          className="w-4 h-4 border-gray-900/10 bg-gray-900/5 transition-all hover:before:opacity-0"
          checked={exit.StableID === prefs.ExitNodeID}
          disabled={!exit.Online}
          onChange={() => {
            GreetService.Send(
              JSON.stringify({ cmd: "exitNode", data: exit.StableID })
            );
          }}
          label={
            <Typography color="blue-gray" className="text-sm">
              {exit.ComputedName}
            </Typography>
          }
        />
      ))}
    </div>
  );
}

export default ExitNode;
