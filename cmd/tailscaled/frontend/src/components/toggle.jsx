import { Switch, Typography } from "@material-tailwind/react";
import * as GreetService from "/bindings/main/GreetService.js";

function Toggle({ state }) {
  const handleToggle = () => {
    console.log("Toggle clicked, current state: ", state);
    switch (state) {
      case 0:
      case 2:
      case 3:
      case 4:
        GreetService.Send(JSON.stringify({ cmd: "up" }));
        break;
      case 6:
        GreetService.Send(JSON.stringify({ cmd: "down" }));
        break;
      default:
        console.log("Invalid state");
    }
  };
  return (
    <div className="flex flex-col items-center">
      <Switch
        ripple={false}
        className="h-full w-full checked:bg-[#2ec946]"
        containerProps={{
          className: "w-11 h-6",
        }}
        circleProps={{
          className: "before:hidden left-0.5 border-none",
        }}
        checked={state >= 5 ? true : false}
        onClick={() => 0}
        onChange={handleToggle}
      />
      <Typography variant="small" color="gray" className="font-normal">
        {state >= 5 ? "已连接" : "未连接"}
      </Typography>
    </div>
  );
}
export default Toggle;
