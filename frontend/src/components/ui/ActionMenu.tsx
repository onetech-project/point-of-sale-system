import React, { useEffect, useRef, useState } from 'react';
import { MoreVertical } from 'lucide-react';

export interface ActionMenuItem {
  label: string;
  onSelect: () => void;
  disabled?: boolean;
  variant?: 'default' | 'danger';
}

interface ActionMenuProps {
  ariaLabel: string;
  items: ActionMenuItem[];
  align?: 'left' | 'right';
  className?: string;
  buttonClassName?: string;
  menuClassName?: string;
}

const itemClassName = (item: ActionMenuItem): string => {
  if (item.disabled) {
    return 'cursor-not-allowed text-gray-300';
  }

  if (item.variant === 'danger') {
    return 'text-red-700 hover:bg-red-50 hover:text-red-800';
  }

  return 'text-gray-700 hover:bg-gray-50 hover:text-gray-900';
};

export const ActionMenu: React.FC<ActionMenuProps> = ({
  ariaLabel,
  items,
  align = 'right',
  className = '',
  buttonClassName = '',
  menuClassName = '',
}) => {
  const [open, setOpen] = useState(false);
  const menuRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    if (!open) return;

    const handlePointerDown = (event: MouseEvent) => {
      if (!menuRef.current?.contains(event.target as Node)) {
        setOpen(false);
      }
    };

    const handleKeyDown = (event: KeyboardEvent) => {
      if (event.key === 'Escape') {
        setOpen(false);
      }
    };

    document.addEventListener('mousedown', handlePointerDown);
    document.addEventListener('keydown', handleKeyDown);
    return () => {
      document.removeEventListener('mousedown', handlePointerDown);
      document.removeEventListener('keydown', handleKeyDown);
    };
  }, [open]);

  return (
    <div
      ref={menuRef}
      className={`relative inline-block text-left ${className}`}
      onClick={event => event.stopPropagation()}
    >
      <button
        type="button"
        onClick={event => {
          event.stopPropagation();
          setOpen(current => !current);
        }}
        className={`inline-flex h-8 w-8 items-center justify-center rounded-md border border-gray-200 text-gray-600 hover:bg-gray-50 hover:text-gray-900 focus:outline-none focus:ring-2 focus:ring-blue-500 ${buttonClassName}`}
        aria-label={ariaLabel}
        aria-expanded={open}
        aria-haspopup="menu"
      >
        <MoreVertical className="h-4 w-4" aria-hidden="true" />
      </button>
      {open && (
        <div
          className={`absolute grid z-30 mt-2 w-48 rounded-md border border-gray-200 bg-white py-1 text-left shadow-lg ${
            align === 'right' ? 'right-0' : 'left-0'
          } ${menuClassName}`}
          role="menu"
        >
          {items.map(item => (
            <button
              key={item.label}
              type="button"
              role="menuitem"
              onClick={event => {
                event.stopPropagation();
                if (item.disabled) return;

                setOpen(false);
                item.onSelect();
              }}
              disabled={item.disabled}
              className={`w-full px-4 py-2 text-left text-sm ${itemClassName(item)}`}
            >
              {item.label}
            </button>
          ))}
        </div>
      )}
    </div>
  );
};

export default ActionMenu;
