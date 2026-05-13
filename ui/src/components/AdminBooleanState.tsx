import React from "react";
import { Check } from "react-feather";

export default function AdminBooleanState({
  value,
}: {
  value?: boolean;
}): React.ReactElement {
  const on = !!value;
  return (
    <span
      className={
        on
          ? "admin-boolean-state admin-boolean-state--true"
          : "admin-boolean-state admin-boolean-state--false"
      }
      aria-hidden
    >
      {on ? (
        <Check
          className="admin-boolean-state__icon"
          size={16}
          strokeWidth={3}
          color="#fff"
          style={{
            width: "16px",
            height: "16px",
            minWidth: "16px",
            minHeight: "16px",
            flexShrink: 0,
            verticalAlign: "middle",
          }}
        />
      ) : null}
    </span>
  );
}
