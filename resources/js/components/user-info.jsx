import { Avatar, AvatarFallback } from "@/components/ui/avatar";
import { useInitials } from "@/hooks/use-initials";

export function UserInfo({ user, showEmail = false }) {
  const getInitials = useInitials();

  return (
    <>
      <Avatar className="h-8 w-8 overflow-hidden rounded-lg">
        <AvatarFallback className="rounded-lg">
          {getInitials(user?.name)}
        </AvatarFallback>
      </Avatar>
      <div className="grid flex-1 text-left text-sm leading-tight">
        <span className="truncate font-medium">{user?.name}</span>
        {showEmail && (
          <span className="text-muted-foreground truncate text-xs">
            {user?.email}
          </span>
        )}
      </div>
    </>
  );
}
