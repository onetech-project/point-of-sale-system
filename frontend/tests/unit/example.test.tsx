import { render, screen } from '@testing-library/react';
import '@testing-library/jest-dom';

// T003: Example React component test
describe('Example Component Test', () => {
  it('should render test component', () => {
    const TestComponent = () => <div>Test Component</div>;
    render(<TestComponent />);
    expect(screen.getByText('Test Component')).toBeInTheDocument();
  });

  it('should handle button click', () => {
    const handleClick = jest.fn();
    const ButtonComponent = () => (
      <button onClick={handleClick}>Click me</button>
    );
    
    render(<ButtonComponent />);
    const button = screen.getByText('Click me');
    button.click();
    
    expect(handleClick).toHaveBeenCalledTimes(1);
  });
});
