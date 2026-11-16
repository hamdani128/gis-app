"use client";

import { createContext, useContext } from "react";
import type { ReactNode } from "react";

export type UserProfile = {
  id: number;
  username: string;
  name: string;
  email: string;
  department: string;
  process_role?: string;
};

const UserProfileContext = createContext<UserProfile | null>(null);
const UpdateUserProfileContext = createContext<((profile: UserProfile) => void) | null>(null);

export function UserProfileProvider({
  profile,
  onUpdate,
  children,
}: {
  profile: UserProfile;
  onUpdate: (profile: UserProfile) => void;
  children: ReactNode;
}) {
  return (
    <UserProfileContext.Provider value={profile}>
      <UpdateUserProfileContext.Provider value={onUpdate}>
        {children}
      </UpdateUserProfileContext.Provider>
    </UserProfileContext.Provider>
  );
}

export function useUserProfile() {
  const profile = useContext(UserProfileContext);
  if (!profile) throw new Error("UserProfileProvider is required");
  return profile;
}

export function useUpdateUserProfile() {
  const update = useContext(UpdateUserProfileContext);
  if (!update) throw new Error("UserProfileProvider is required");
  return update;
}
