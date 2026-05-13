import React from "react";

interface Props {
  text: string;
  title: string;
  onClick: () => void;
  disabled?: boolean;
  className?: string;
}

const IconTextButton: React.FC<Props> = ({
  text,
  title,
  onClick,
  disabled,
  className = "",
}) => {
  return (
    <button
      type="button"
      className={`ms-2 btn btn-light d-flex align-items-center search-panel-outline-btn ${className}`.trim()}
      disabled={disabled}
      onClick={onClick}
      title={title}
    >
      {text}
    </button>
  );
};

export default IconTextButton;
