import React from "react";
import { Typography, Button } from "@material-tailwind/react";
import { CheckIcon, DocumentDuplicateIcon } from "@heroicons/react/24/solid";
import { useCopyToClipboard } from "usehooks-ts";

function NodeDetail({ peer, user }) {
  const [value, copy] = useCopyToClipboard();
  const [copiedv4, setCopiedv4] = React.useState(false);
  const [copiedv6, setCopiedv6] = React.useState(false);
  return (
    <div className="w-full h-full px-4 pt-12 rounded-lg">
      <div className="flex">
        <Typography className="font-normal text-2xl">
          {peer.ComputedName}
        </Typography>
        <div
          className={`h-2 w-2 ${!(peer.Online === false) ? "bg-green-400" : "bg-gray-500"} rounded-full`}
        ></div>
      </div>
      <hr className="my-2 border-blue-gray-300" />
      <div className="flex flex-row w-80">
        <div className="w-30">
          <Typography className="font-normal text-blue-gray-300 text-sm">
            所有者
          </Typography>
          <Typography className="font-normal text-blue-gray-600 text-sm">
            {user.DisplayName}
          </Typography>
          <Typography className="font-normal text-blue-gray-600 text-xs">
            {user.LoginName}
          </Typography>
        </div>
        <div className="ml-10 w-30">
          <Typography className="font-normal text-blue-gray-300 text-sm">
            状态
          </Typography>
          <Typography className="font-normal text-blue-gray-600 text-sm">
            {!(peer.Online === false) ? "在线" : "离线"}
          </Typography>
        </div>
        <div></div>
      </div>
      <div className="flex mt-10">
        <Typography className="font-normal text-2xl">设备详情</Typography>
      </div>
      <hr className="my-2 border-blue-gray-300" />
      <div className="w-full h-full rounded-md border border-gray-300 p-2">
        <div className="flex flex-row w-80">
          <div className="w-30 flex">
            <Typography className="w-40 font-normal text-blue-gray-300 text-sm">
              所有者
            </Typography>
            <Typography className="pl-1 font-normal text-blue-gray-600 text-sm">
              {user.LoginName}
            </Typography>
          </div>
        </div>
        <div className="flex flex-row w-80">
          <div className="w-30 flex">
            <Typography className="w-40 font-normal text-blue-gray-300 text-sm">
              设备名
            </Typography>
            <Typography className="pl-1 font-normal text-blue-gray-600 text-sm">
              {peer.ComputedName}
            </Typography>
          </div>
        </div>
        <div className="flex flex-row w-80">
          <div className="w-30 flex">
            <Typography className="w-40 font-normal text-blue-gray-300 text-sm">
              操作系统 Hostname
            </Typography>
            <Typography className="pl-1 font-normal text-blue-gray-600 text-sm">
              {peer.Hostinfo.Hostname}
            </Typography>
          </div>
        </div>
        <div className="flex flex-row w-80">
          <div className="w-30 flex">
            <Typography className="w-40 font-normal text-blue-gray-300 text-sm">
              操作系统
            </Typography>
            <Typography className="pl-1 font-normal text-blue-gray-600 text-sm">
              {peer.Hostinfo.OS}
            </Typography>
          </div>
        </div>
        <div className="flex flex-row w-80">
          <div className="w-30 flex">
            <Typography className="w-40 font-normal text-blue-gray-300 text-sm">
              创建时间
            </Typography>
            <Typography className="pl-1 font-normal text-blue-gray-600 text-sm">
              {new Date(peer.Created).toLocaleString()}
            </Typography>
          </div>
        </div>
        <div className="flex flex-row w-80">
          <div className="w-30 flex">
            <Typography className="w-40 font-normal text-blue-gray-300 text-sm">
              上次登录时间
            </Typography>
            <Typography className="pl-1 font-normal text-blue-gray-600 text-sm">
              {peer.Online
                ? "当前在线"
                : new Date(peer.LastSeen).toLocaleString()}
            </Typography>
          </div>
        </div>
        <div className="flex flex-row w-80">
          <div className="w-30 flex">
            <Typography className="w-40 font-normal text-blue-gray-300 text-sm">
              IPv4 地址
            </Typography>
            <Button
              className="flex p-1 pl-0 bg-transparent"
              onClick={() => {
                copy(peer.Addresses[0].slice(0, -3));
                setCopiedv4(true);
              }}
              onMouseLeave={() => setCopiedv4(false)}
            >
              <Typography className="pl-1 font-normal text-blue-gray-600 text-sm">
                {peer.Addresses[0].slice(0, -3)}
              </Typography>
              {copiedv4 ? (
                <CheckIcon className="h-4 w-4 text-blue-gray-500" />
              ) : (
                <DocumentDuplicateIcon className="h-4 w-4 text-blue-gray-500" />
              )}
            </Button>
          </div>
        </div>
        <div className="flex flex-row w-80">
          <div className="w-30 flex">
            <Typography className="w-40 font-normal text-blue-gray-300 text-sm">
              IPv6 地址
            </Typography>
            <Button
              className="flex p-1 pl-0 bg-transparent"
              onClick={() => {
                copy(peer.Addresses[1].slice(0, -4));
                setCopiedv6(true);
              }}
              onMouseLeave={() => setCopiedv6(false)}
            >
              <Typography className="pl-1 font-normal text-blue-gray-600 text-sm">
                {processIPv6(peer.Addresses[1])}
              </Typography>
              {copiedv6 ? (
                <CheckIcon className="h-4 w-4 text-blue-gray-500" />
              ) : (
                <DocumentDuplicateIcon className="h-4 w-4 text-blue-gray-500" />
              )}
            </Button>
          </div>
        </div>
        <div className="flex flex-row w-80">
          <div className="w-30 flex">
            <Typography className="w-40 font-normal text-blue-gray-300 text-sm">
              域名
            </Typography>
            <Typography className="pl-1 font-normal text-blue-gray-600 text-sm">
              {peer.Name}
            </Typography>
          </div>
        </div>
      </div>
    </div>
  );
}

function processIPv6(address) {
  if (address.length > 20) {
    return address.slice(0, 8) + "..." + address.slice(-12, -4);
  } else {
    return address;
  }
}
export default NodeDetail;
