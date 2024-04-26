import { Avatar, Typography } from "@material-tailwind/react";

export function AvatarWithText({ prefs }) {
  return (
    <div className="flex items-center gap-1">
      <Avatar src={prefs.Config.UserProfile.ProfilePicURL} alt="avatar" />
      <div>
        <Typography variant="h6">
          {prefs.Config.UserProfile.DisplayName}
        </Typography>
        <Typography variant="small" color="gray" className="font-normal">
          {prefs.Config.UserProfile.LoginName}
        </Typography>
      </div>
    </div>
  );
}

export default AvatarWithText;
