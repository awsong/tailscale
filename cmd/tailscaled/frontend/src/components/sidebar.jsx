import React from "react";
import {
  Button,
  Card,
  Typography,
  List,
  ListItem,
  ListItemPrefix,
  ListItemSuffix,
  Popover,
  PopoverContent,
  PopoverHandler,
  Chip,
  Accordion,
  AccordionHeader,
  AccordionBody,
} from "@material-tailwind/react";
import AvatarWithText from "./avatar";
import Toggle from "./toggle";
import NodeDetail from "./nodedetail";
import ExitNode from "./exitnode";
import {
  PresentationChartBarIcon,
  UserCircleIcon,
  Cog6ToothIcon,
  InboxIcon,
  PowerIcon,
} from "@heroicons/react/24/solid";
import { ChevronRightIcon, ChevronDownIcon } from "@heroicons/react/24/outline";
import {
  IconFinder,
  IconLinux,
  IconWindows,
  IconIos,
  IconApple,
  IconAndroid,
} from "./icons";
import * as GreetService from "/bindings/main/GreetService.js";
function test(netMap) {
  if (!netMap) {
    return [];
  }

  const userMap = new Map();

  netMap.Peers.forEach((peer) => {
    if (userMap.has(peer.User)) {
      userMap.get(peer.User).push(peer);
    } else {
      userMap.set(peer.User, [peer]);
    }
  });

  // Convert the map to a two-level array
  const groupedPeers = Array.from(userMap).map(([userID, userList]) => {
    let name;
    if (userID == netMap.SelfNode.User) {
      name = "我的设备";
    } else {
      name = netMap.UserProfiles[userID].DisplayName + "的设备";
    }
    return {
      Name: name,
      Peers: userList,
    };
  });

  return groupedPeers.reverse();
}
function OSIcon({ peer }) {
  switch (peer.Hostinfo.OS) {
    case "linux":
      return (
        <IconLinux
          className={`ml-2 ${peer.Online ? "text-green-400" : "text-gray-400"}`}
        />
      );
    case "windows":
      return (
        <IconWindows
          className={`ml-2 ${peer.Online ? "text-green-400" : "text-gray-400"}`}
        />
      );
    case "macOS":
      return (
        <IconFinder
          className={`ml-2 ${peer.Online ? "text-green-400" : "text-gray-400"}`}
        />
      );
    case "android":
      return (
        <IconAndroid
          className={`ml-2 ${peer.Online ? "text-green-400" : "text-gray-400"}`}
        />
      );
    case "iOS":
      return (
        <IconApple
          className={`ml-2 ${peer.Online ? "text-green-400" : "text-gray-400"}`}
        />
      );
  }
}
function SidebarWithContentSeparator({ prefs, netMap, state }) {
  const [open, setOpen] = React.useState(-1);

  const handleOpen = (value) => {
    setOpen(open === value ? -1 : value);
  };

  const [detailPeer, setDetailPeer] = React.useState(
    netMap ? netMap.SelfNode : null
  );

  return (
    <>
      <Card className="h-full w-full max-w-[20rem] p-4 shadow-xl shadow-blue-gray-900/5">
        <div className="b-2 p-4 flex justify-between items-center">
          {prefs.Config && <AvatarWithText prefs={prefs} />}
          <Toggle state={state} />
        </div>
        <hr className="my-2 border-blue-gray-50" />
        <List>
          {netMap && (
            <ListItem>
              <ListItemPrefix>
                <UserCircleIcon className="h-5 w-5" />
              </ListItemPrefix>
              {"本机: " +
                netMap.SelfNode.ComputedName +
                "(" +
                netMap.SelfNode.Addresses[0] +
                ")"}
            </ListItem>
          )}
          <Accordion open={true}>
            <AccordionHeader className="border-b-0 p-3">
              <ListItemPrefix>
                <PresentationChartBarIcon className="h-5 w-5" />
              </ListItemPrefix>
              <Typography color="blue-gray" className="mr-auto font-normal">
                在网设备
              </Typography>
            </AccordionHeader>
            <AccordionBody className="py-0">
              {test(netMap).map((group, index) => (
                <Accordion
                  open={open === index}
                  className="pl-3"
                  icon={
                    <ChevronDownIcon
                      onClick={() => handleOpen(index)}
                      strokeWidth={2.5}
                      className={`h-3 w-5 ${open === index ? "rotate-180" : ""}`}
                    />
                  }
                >
                  <ListItem className="p-0">
                    <AccordionHeader className="border-b-0 p-0">
                      <Typography
                        color="blue-gray"
                        className="mr-auto font-normal pl-10"
                      >
                        {group.Name}
                      </Typography>
                    </AccordionHeader>
                  </ListItem>
                  <AccordionBody className="py-0 max-h-40 overflow-y-auto overflow-x-hidden">
                    <List className="p-0">
                      {group.Peers.map((peer, key) => (
                        <ListItem
                          onClick={() => setDetailPeer(peer)}
                          className={`text-sm ml-4 pl-9 py-0`}
                        >
                          {peer.ComputedName}
                          <OSIcon peer={peer} />
                        </ListItem>
                      ))}
                    </List>
                  </AccordionBody>
                </Accordion>
              ))}
            </AccordionBody>
          </Accordion>
          <hr className="my-2 border-blue-gray-50" />
          <Popover placement="right-end">
            <PopoverHandler>
              <ListItem>
                <ListItemPrefix>
                  <UserCircleIcon className="h-5 w-5" />
                </ListItemPrefix>
                出口节点
              </ListItem>
            </PopoverHandler>
            <PopoverContent className="m-5">
              {netMap ? <ExitNode netMap={netMap} prefs={prefs} /> : <></>}
            </PopoverContent>
          </Popover>
          <ListItem>
            <ListItemPrefix>
              <Cog6ToothIcon className="h-5 w-5" />
            </ListItemPrefix>
            Settings
          </ListItem>
          <ListItem
            onClick={() => GreetService.Send(JSON.stringify({ cmd: "logout" }))}
          >
            <ListItemPrefix>
              <PowerIcon className="h-5 w-5" />
            </ListItemPrefix>
            登出
          </ListItem>
        </List>
      </Card>
      {detailPeer && (
        <NodeDetail
          peer={detailPeer}
          user={netMap.UserProfiles[detailPeer.User]}
        />
      )}
    </>
  );
}
export default SidebarWithContentSeparator;
