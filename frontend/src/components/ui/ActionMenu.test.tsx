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
