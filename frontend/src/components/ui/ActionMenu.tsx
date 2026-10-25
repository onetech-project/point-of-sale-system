import React, { useCallback, useEffect, useLayoutEffect, useRef, useState } from 'react';
import { createPortal } from 'react-dom';
import { MoreVertical, type LucideIcon } from 'lucide-react';

export interface ActionMenuItem {
  label: string;
  onSelect: () => void;
  disabled?: boolean;
  variant?: 'default' | 'danger';
  icon?: LucideIcon;
  /** Draws a divider above this item, to separate groups of actions */
  separatorBefore?: boolean;
}

interface ActionMenuProps {
  ariaLabel: string;
  items: ActionMenuItem[];
  align?: 'left' | 'right';
  className?: string;
  buttonClassName?: string;
  menuClassName?: string;
}

// Gap between the trigger and the menu, and the minimum distance kept from the viewport edges
const MENU_OFFSET = 8;
const VIEWPORT_MARGIN = 8;

interface MenuPosition {
  top: number;
  left: number;
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

/**
 * Places the menu below the trigger, or above it when there is no room below, aligned to the
 * trigger's right or left edge and kept inside the viewport
 */
const computeMenuPosition = (trigger: DOMRect, menu: DOMRect, align: 'left' | 'right'): MenuPosition => {
  const viewportWidth = window.innerWidth;
  const viewportHeight = window.innerHeight;

  let top = trigger.bottom + MENU_OFFSET;
  const fitsBelow = top + menu.height <= viewportHeight - VIEWPORT_MARGIN;
  const fitsAbove = trigger.top - MENU_OFFSET - menu.height >= VIEWPORT_MARGIN;
  if (!fitsBelow && fitsAbove) {
    top = trigger.top - MENU_OFFSET - menu.height;
  }

  const preferredLeft = align === 'right' ? trigger.right - menu.width : trigger.left;
  const maxLeft = Math.max(VIEWPORT_MARGIN, viewportWidth - menu.width - VIEWPORT_MARGIN);
  const left = Math.min(Math.max(preferredLeft, VIEWPORT_MARGIN), maxLeft);

  return { top, left };
};

/**
 * Three-dot action menu. The menu is portaled to document.body with fixed positioning so it is
 * not clipped by scrolling or overflow-hidden containers such as table wrappers.
 */
export const ActionMenu: React.FC<ActionMenuProps> = ({
  ariaLabel,
  items,
  align = 'right',
  className = '',
  buttonClassName = '',
  menuClassName = '',
}) => {
  const [open, setOpen] = useState(false);
  const [position, setPosition] = useState<MenuPosition | null>(null);
  const triggerRef = useRef<HTMLButtonElement>(null);
  const menuRef = useRef<HTMLDivElement>(null);

  const close = useCallback((restoreFocus = false) => {
    setOpen(false);
    setPosition(null);
    if (restoreFocus) {
      triggerRef.current?.focus();
    }
  }, []);

  const updatePosition = useCallback(() => {
    const trigger = triggerRef.current;
    const menu = menuRef.current;
    if (!trigger || !menu) return;
    setPosition(computeMenuPosition(trigger.getBoundingClientRect(), menu.getBoundingClientRect(), align));
  }, [align]);

  // Measure before paint so the menu never flashes at the wrong place
  useLayoutEffect(() => {
    if (open) {
      updatePosition();
    }
  }, [open, items.length, updatePosition]);

  useEffect(() => {
    if (!open) return;

    const handlePointerDown = (event: MouseEvent) => {
      const target = event.target as Node;
      // The menu lives in a portal, so it is outside the trigger's DOM subtree
      if (!triggerRef.current?.contains(target) && !menuRef.current?.contains(target)) {
        close();
      }
    };

    const handleKeyDown = (event: KeyboardEvent) => {
      if (event.key === 'Escape') {
        close(true);
      }
    };

    // Capture scrolls from any scrollable ancestor, not only the window
    window.addEventListener('scroll', updatePosition, true);
    window.addEventListener('resize', updatePosition);
    document.addEventListener('mousedown', handlePointerDown);
    document.addEventListener('keydown', handleKeyDown);
    return () => {
      window.removeEventListener('scroll', updatePosition, true);
      window.removeEventListener('resize', updatePosition);
      document.removeEventListener('mousedown', handlePointerDown);
      document.removeEventListener('keydown', handleKeyDown);
    };
  }, [open, close, updatePosition]);

  // The portal sits at the end of the document, outside the natural tab order, so move focus into
  // the menu when it opens and hand it back to the trigger when tabbing out
  useEffect(() => {
    if (!open || !position) return;
    menuRef.current?.querySelector<HTMLButtonElement>('[role="menuitem"]:not(:disabled)')?.focus();
  }, [open, position]);

  const handleMenuKeyDown = (event: React.KeyboardEvent<HTMLDivElement>) => {
    if (event.key === 'Tab') {
      event.preventDefault();
      close(true);
    }
  };

  const menu = open
    ? createPortal(
        <div
          ref={menuRef}
          role="menu"
          aria-label={ariaLabel}
          // React events bubble through portals to the trigger's ancestors, such as clickable table rows
          onClick={event => event.stopPropagation()}
          onKeyDown={handleMenuKeyDown}
          style={{
            top: position?.top ?? 0,
            left: position?.left ?? 0,
            visibility: position ? 'visible' : 'hidden',
          }}
          className={`fixed z-[60] grid w-48 rounded-md border border-gray-200 bg-white py-1 text-left shadow-lg ${menuClassName}`}
        >
          {items.map((item, index) => {
            const Icon = item.icon;
            return (
              <React.Fragment key={item.label}>
                {item.separatorBefore && index > 0 && (
                  <div role="separator" className="my-1 border-t border-gray-200" />
                )}
                <button
                  type="button"
                  role="menuitem"
                  onClick={event => {
                    event.stopPropagation();
                    if (item.disabled) return;

                    close();
                    item.onSelect();
                  }}
                  disabled={item.disabled}
                  className={`flex w-full items-center gap-2 px-4 py-2 text-left text-sm focus:bg-gray-50 focus:outline-none ${itemClassName(item)}`}
                >
                  {Icon && <Icon className="h-4 w-4 shrink-0" aria-hidden="true" />}
                  {item.label}
                </button>
              </React.Fragment>
            );
          })}
        </div>,
        document.body
      )
    : null;

  return (
    <div className={`relative inline-block text-left ${className}`} onClick={event => event.stopPropagation()}>
      <button
        ref={triggerRef}
        type="button"
        onClick={event => {
          event.stopPropagation();
          if (open) {
            close();
          } else {
            setOpen(true);
          }
        }}
        className={`inline-flex h-8 w-8 items-center justify-center rounded-md border border-gray-200 text-gray-600 hover:bg-gray-50 hover:text-gray-900 focus:outline-none focus:ring-2 focus:ring-blue-500 ${buttonClassName}`}
        aria-label={ariaLabel}
        aria-expanded={open}
        aria-haspopup="menu"
      >
        <MoreVertical className="h-4 w-4" aria-hidden="true" />
      </button>
      {menu}
    </div>
  );
};

export default ActionMenu;
