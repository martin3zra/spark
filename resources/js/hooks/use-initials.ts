import { useCallback } from "react";

export function useInitials() {
  return useCallback((fullName = ""): string => {
    const names = fullName.trim().split(" ").filter(Boolean);
    if (names.length === 0) return "";
    if (names.length === 1) return names[0].charAt(0).toUpperCase();
    return `${names[0].charAt(0)}${names[names.length - 1].charAt(0)}`.toUpperCase();
  }, []);
}
