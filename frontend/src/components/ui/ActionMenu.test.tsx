import React from 'react';
import { fireEvent, render, screen } from '@testing-library/react';
import '@testing-library/jest-dom';
import ActionMenu from './ActionMenu';

describe('ActionMenu', () => {
  it('opens and closes from the trigger', () => {
    render(
      <ActionMenu
        ariaLabel="Open actions"
        items={[{ label: 'View Details', onSelect: jest.fn() }]}
      />
    );

    const trigger = screen.getByLabelText('Open actions');
    fireEvent.click(trigger);
    expect(screen.getByRole('menu')).toBeInTheDocument();

    fireEvent.click(trigger);
    expect(screen.queryByRole('menu')).not.toBeInTheDocument();
  });

  it('closes on outside click and Escape', () => {
    render(
      <ActionMenu
        ariaLabel="Open actions"
        items={[{ label: 'View Details', onSelect: jest.fn() }]}
      />
    );

    fireEvent.click(screen.getByLabelText('Open actions'));
    fireEvent.mouseDown(document.body);
    expect(screen.queryByRole('menu')).not.toBeInTheDocument();

    fireEvent.click(screen.getByLabelText('Open actions'));
    fireEvent.keyDown(document, { key: 'Escape' });
    expect(screen.queryByRole('menu')).not.toBeInTheDocument();
  });

  it('calls enabled actions and closes the menu', () => {
    const onSelect = jest.fn();
    render(
      <ActionMenu
        ariaLabel="Open actions"
        items={[{ label: 'Download Invoice', onSelect }]}
      />
    );

    fireEvent.click(screen.getByLabelText('Open actions'));
    fireEvent.click(screen.getByRole('menuitem', { name: 'Download Invoice' }));

    expect(onSelect).toHaveBeenCalledTimes(1);
    expect(screen.queryByRole('menu')).not.toBeInTheDocument();
  });

  it('renders a separator between groups but not before the first item', () => {
    render(
      <ActionMenu
        ariaLabel="Open actions"
        items={[
          { label: 'Edit Order', onSelect: jest.fn(), separatorBefore: true },
          { label: 'Download Invoice', onSelect: jest.fn(), separatorBefore: true },
        ]}
      />
    );

    fireEvent.click(screen.getByLabelText('Open actions'));

    const menu = screen.getByRole('menu');
    expect(screen.getAllByRole('separator')).toHaveLength(1);
    expect(menu.firstElementChild).toHaveAttribute('role', 'menuitem');
  });

  it('mounts the menu at the document root so overflow containers do not clip it', () => {
    const { container } = render(
      <div style={{ overflow: 'hidden' }}>
        <ActionMenu ariaLabel="Open actions" items={[{ label: 'View Details', onSelect: jest.fn() }]} />
      </div>
    );

    fireEvent.click(screen.getByLabelText('Open actions'));

    const menu = screen.getByRole('menu');
    expect(container.contains(menu)).toBe(false);
    expect(menu.parentElement).toBe(document.body);
    expect(menu).toHaveClass('fixed');
  });

  it('stays open when pressing inside the portaled menu', () => {
    render(
      <ActionMenu ariaLabel="Open actions" items={[{ label: 'View Details', onSelect: jest.fn() }]} />
    );

    fireEvent.click(screen.getByLabelText('Open actions'));
    fireEvent.mouseDown(screen.getByRole('menuitem', { name: 'View Details' }));

    expect(screen.getByRole('menu')).toBeInTheDocument();
  });

  it('does not trigger click handlers of the row it sits in', () => {
    const onRowClick = jest.fn();
    const onSelect = jest.fn();
    render(
      <div onClick={onRowClick}>
        <ActionMenu ariaLabel="Open actions" items={[{ label: 'Download Invoice', onSelect }]} />
      </div>
    );

    fireEvent.click(screen.getByLabelText('Open actions'));
    fireEvent.click(screen.getByRole('menuitem', { name: 'Download Invoice' }));

    expect(onSelect).toHaveBeenCalledTimes(1);
    expect(onRowClick).not.toHaveBeenCalled();
  });

  it('flips above the trigger when there is no room below', () => {
    const originalHeight = window.innerHeight;
    Object.defineProperty(window, 'innerHeight', { configurable: true, value: 300 });
    const rectSpy = jest
      .spyOn(HTMLElement.prototype, 'getBoundingClientRect')
      .mockImplementation(function (this: HTMLElement) {
        const box = this.getAttribute('role') === 'menu'
          ? { top: 0, bottom: 120, left: 0, right: 192, width: 192, height: 120 }
          : { top: 260, bottom: 292, left: 400, right: 432, width: 32, height: 32 };
        return { ...box, x: box.left, y: box.top, toJSON: () => box } as DOMRect;
      });

    try {
      render(<ActionMenu ariaLabel="Open actions" items={[{ label: 'View Details', onSelect: jest.fn() }]} />);
      fireEvent.click(screen.getByLabelText('Open actions'));

      const menu = screen.getByRole('menu');
      // 260 (trigger top) - 8 (offset) - 120 (menu height)
      expect(menu.style.top).toBe('132px');
      // Right-aligned with the trigger: 432 - 192
      expect(menu.style.left).toBe('240px');
      expect(menu.style.visibility).toBe('visible');
    } finally {
      rectSpy.mockRestore();
      Object.defineProperty(window, 'innerHeight', { configurable: true, value: originalHeight });
    }
  });

  it('focuses the first enabled item and returns focus to the trigger on Escape', () => {
    render(
      <ActionMenu
        ariaLabel="Open actions"
        items={[
          { label: 'Download Receipt', onSelect: jest.fn(), disabled: true },
          { label: 'Download Invoice', onSelect: jest.fn() },
        ]}
      />
    );

    const trigger = screen.getByLabelText('Open actions');
    fireEvent.click(trigger);
    expect(screen.getByRole('menuitem', { name: 'Download Invoice' })).toHaveFocus();

    fireEvent.keyDown(document, { key: 'Escape' });
    expect(screen.queryByRole('menu')).not.toBeInTheDocument();
    expect(trigger).toHaveFocus();
  });

  it('does not call disabled actions', () => {
    const onSelect = jest.fn();
    render(
      <ActionMenu
        ariaLabel="Open actions"
        items={[{ label: 'Download Receipt', onSelect, disabled: true }]}
      />
    );

    fireEvent.click(screen.getByLabelText('Open actions'));
    fireEvent.click(screen.getByRole('menuitem', { name: 'Download Receipt' }));

    expect(onSelect).not.toHaveBeenCalled();
    expect(screen.getByRole('menu')).toBeInTheDocument();
  });
});
