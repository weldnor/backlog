// The handful of inline icons the old UI drew, carried over verbatim. Each is
// a currentColor stroke icon sized by its caller.

import type { ReactNode } from "react";

interface IconProps {
  size?: number;
}

function Svg({ size = 14, children }: IconProps & { children: ReactNode }) {
  return (
    <svg
      width={size}
      height={size}
      viewBox="0 0 24 24"
      fill="none"
      stroke="currentColor"
      strokeWidth={2}
      strokeLinecap="round"
      strokeLinejoin="round"
    >
      {children}
    </svg>
  );
}

export const SearchIcon = ({ size = 15 }: IconProps) => (
  <Svg size={size}>
    <circle cx="11" cy="11" r="8" />
    <path d="m21 21-4.3-4.3" />
  </Svg>
);

export const PlusIcon = ({ size = 14 }: IconProps) => (
  <Svg size={size}>
    <path d="M5 12h14" />
    <path d="M12 5v14" />
  </Svg>
);

export const SaveIcon = ({ size = 14 }: IconProps) => (
  <Svg size={size}>
    <path d="M19 21H5a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h11l5 5v11a2 2 0 0 1-2 2z" />
    <path d="M17 21v-8H7v8" />
    <path d="M7 3v5h8" />
  </Svg>
);

export const EditIcon = ({ size = 14 }: IconProps) => (
  <Svg size={size}>
    <path d="M12 20h9" />
    <path d="M16.5 3.5a2.121 2.121 0 0 1 3 3L7 19l-4 1 1-4Z" />
  </Svg>
);

export const CloseIcon = ({ size = 16 }: IconProps) => (
  <Svg size={size}>
    <path d="M18 6 6 18" />
    <path d="m6 6 12 12" />
  </Svg>
);
