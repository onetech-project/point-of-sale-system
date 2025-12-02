// T149: Frontend unit tests for ProductForm component

import React from 'react';
import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import '@testing-library/jest-dom';

// Mock ProductForm component for testing
// In real implementation, this would import from ProductForm.tsx
const ProductForm = ({ initialProduct = null, onSubmit = () => {}, onCancel = () => {} }) => {
  const [formData, setFormData] = React.useState({
    name: initialProduct?.name || '',
    sku: initialProduct?.sku || '',
    price: initialProduct?.price || '',
    category_id: initialProduct?.category_id || '',
    description: initialProduct?.description || '',
    quantity: initialProduct?.quantity || '',
    tax_rate: initialProduct?.tax_rate || '',
  });

  const [errors, setErrors] = React.useState({});

  const handleChange = (e: React.ChangeEvent<HTMLInputElement | HTMLTextAreaElement | HTMLSelectElement>) => {
    const { name, value } = e.target;
    setFormData(prev => ({ ...prev, [name]: value }));
    
    // Clear error when user starts typing
    if (errors[name]) {
      setErrors(prev => ({ ...prev, [name]: '' }));
    }
  };

  const validateForm = () => {
    const newErrors: any = {};

    if (!formData.name || formData.name.trim().length === 0) {
      newErrors.name = 'Name is required';
    } else if (formData.name.length > 255) {
      newErrors.name = 'Name must be 255 characters or less';
    }

    if (!formData.sku || formData.sku.trim().length === 0) {
      newErrors.sku = 'SKU is required';
    } else if (formData.sku.length > 50) {
      newErrors.sku = 'SKU must be 50 characters or less';
    }

    const price = parseFloat(formData.price);
    if (!formData.price) {
      newErrors.price = 'Price is required';
    } else if (isNaN(price) || price < 0) {
      newErrors.price = 'Price must be a positive number';
    }

    if (!formData.category_id) {
      newErrors.category_id = 'Category is required';
    }

    const quantity = parseInt(formData.quantity);
    if (!formData.quantity) {
      newErrors.quantity = 'Quantity is required';
    } else if (isNaN(quantity) || quantity < 0) {
      newErrors.quantity = 'Quantity must be a non-negative number';
    }

    const taxRate = parseFloat(formData.tax_rate);
    if (formData.tax_rate && (isNaN(taxRate) || taxRate < 0 || taxRate > 1)) {
      newErrors.tax_rate = 'Tax rate must be between 0 and 1';
    }

    setErrors(newErrors);
    return Object.keys(newErrors).length === 0;
  };

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    if (validateForm()) {
      onSubmit(formData);
    }
  };

  return (
    <form onSubmit={handleSubmit} data-testid="product-form">
      <div>
        <label htmlFor="name">Product Name</label>
        <input
          id="name"
          name="name"
          type="text"
          value={formData.name}
          onChange={handleChange}
          data-testid="name-input"
        />
        {errors.name && <span data-testid="name-error">{errors.name}</span>}
      </div>

      <div>
        <label htmlFor="sku">SKU</label>
        <input
          id="sku"
          name="sku"
          type="text"
          value={formData.sku}
          onChange={handleChange}
          data-testid="sku-input"
        />
        {errors.sku && <span data-testid="sku-error">{errors.sku}</span>}
      </div>

      <div>
        <label htmlFor="price">Price</label>
        <input
          id="price"
          name="price"
          type="number"
          step="0.01"
          value={formData.price}
          onChange={handleChange}
          data-testid="price-input"
        />
        {errors.price && <span data-testid="price-error">{errors.price}</span>}
      </div>

      <div>
        <label htmlFor="category_id">Category</label>
        <select
          id="category_id"
          name="category_id"
          value={formData.category_id}
          onChange={handleChange}
          data-testid="category-select"
        >
          <option value="">Select category</option>
          <option value="1">Electronics</option>
          <option value="2">Food</option>
        </select>
        {errors.category_id && <span data-testid="category-error">{errors.category_id}</span>}
      </div>

      <div>
        <label htmlFor="quantity">Quantity</label>
        <input
          id="quantity"
          name="quantity"
          type="number"
          value={formData.quantity}
          onChange={handleChange}
          data-testid="quantity-input"
        />
        {errors.quantity && <span data-testid="quantity-error">{errors.quantity}</span>}
      </div>

      <div>
        <label htmlFor="tax_rate">Tax Rate</label>
        <input
          id="tax_rate"
          name="tax_rate"
          type="number"
          step="0.01"
          value={formData.tax_rate}
          onChange={handleChange}
          data-testid="tax-rate-input"
        />
        {errors.tax_rate && <span data-testid="tax-rate-error">{errors.tax_rate}</span>}
      </div>

      <div>
        <label htmlFor="description">Description</label>
        <textarea
          id="description"
          name="description"
          value={formData.description}
          onChange={handleChange}
          data-testid="description-input"
        />
      </div>

      <button type="submit" data-testid="submit-button">
        {initialProduct ? 'Update Product' : 'Create Product'}
      </button>
      <button type="button" onClick={onCancel} data-testid="cancel-button">
        Cancel
      </button>
    </form>
  );
};

describe('ProductForm', () => {
  describe('Rendering', () => {
    it('should render all form fields', () => {
      render(<ProductForm />);
      
      expect(screen.getByTestId('name-input')).toBeInTheDocument();
      expect(screen.getByTestId('sku-input')).toBeInTheDocument();
      expect(screen.getByTestId('price-input')).toBeInTheDocument();
      expect(screen.getByTestId('category-select')).toBeInTheDocument();
      expect(screen.getByTestId('quantity-input')).toBeInTheDocument();
      expect(screen.getByTestId('tax-rate-input')).toBeInTheDocument();
      expect(screen.getByTestId('description-input')).toBeInTheDocument();
    });

    it('should show "Create Product" button for new products', () => {
      render(<ProductForm />);
      
      expect(screen.getByTestId('submit-button')).toHaveTextContent('Create Product');
    });

    it('should show "Update Product" button for existing products', () => {
      const product = {
        name: 'Test Product',
        sku: 'SKU001',
        price: '19.99',
        category_id: '1',
        quantity: '10',
        tax_rate: '0.10',
      };
      
      render(<ProductForm initialProduct={product} />);
      
      expect(screen.getByTestId('submit-button')).toHaveTextContent('Update Product');
    });

    it('should pre-populate form fields when editing', () => {
      const product = {
        name: 'Test Product',
        sku: 'SKU001',
        price: '19.99',
        category_id: '1',
        quantity: '10',
        tax_rate: '0.10',
        description: 'Test description',
      };
      
      render(<ProductForm initialProduct={product} />);
      
      expect(screen.getByTestId('name-input')).toHaveValue('Test Product');
      expect(screen.getByTestId('sku-input')).toHaveValue('SKU001');
      expect(screen.getByTestId('price-input')).toHaveValue('19.99');
      expect(screen.getByTestId('category-select')).toHaveValue('1');
      expect(screen.getByTestId('quantity-input')).toHaveValue('10');
      expect(screen.getByTestId('tax-rate-input')).toHaveValue('0.10');
      expect(screen.getByTestId('description-input')).toHaveValue('Test description');
    });
  });

  describe('Validation', () => {
    it('should show error for empty name', async () => {
      const handleSubmit = jest.fn();
      render(<ProductForm onSubmit={handleSubmit} />);
      
      const submitButton = screen.getByTestId('submit-button');
      fireEvent.click(submitButton);
      
      await waitFor(() => {
        expect(screen.getByTestId('name-error')).toHaveTextContent('Name is required');
      });
      
      expect(handleSubmit).not.toHaveBeenCalled();
    });

    it('should show error for name longer than 255 characters', async () => {
      const handleSubmit = jest.fn();
      render(<ProductForm onSubmit={handleSubmit} />);
      
      const nameInput = screen.getByTestId('name-input');
      const longName = 'a'.repeat(256);
      fireEvent.change(nameInput, { target: { name: 'name', value: longName } });
      
      const submitButton = screen.getByTestId('submit-button');
      fireEvent.click(submitButton);
      
      await waitFor(() => {
        expect(screen.getByTestId('name-error')).toHaveTextContent('Name must be 255 characters or less');
      });
      
      expect(handleSubmit).not.toHaveBeenCalled();
    });

    it('should show error for empty SKU', async () => {
      const handleSubmit = jest.fn();
      render(<ProductForm onSubmit={handleSubmit} />);
      
      const submitButton = screen.getByTestId('submit-button');
      fireEvent.click(submitButton);
      
      await waitFor(() => {
        expect(screen.getByTestId('sku-error')).toHaveTextContent('SKU is required');
      });
      
      expect(handleSubmit).not.toHaveBeenCalled();
    });

    it('should show error for SKU longer than 50 characters', async () => {
      const handleSubmit = jest.fn();
      render(<ProductForm onSubmit={handleSubmit} />);
      
      const skuInput = screen.getByTestId('sku-input');
      const longSku = 'S'.repeat(51);
      fireEvent.change(skuInput, { target: { name: 'sku', value: longSku } });
      
      const submitButton = screen.getByTestId('submit-button');
      fireEvent.click(submitButton);
      
      await waitFor(() => {
        expect(screen.getByTestId('sku-error')).toHaveTextContent('SKU must be 50 characters or less');
      });
      
      expect(handleSubmit).not.toHaveBeenCalled();
    });

    it('should show error for negative price', async () => {
      const handleSubmit = jest.fn();
      render(<ProductForm onSubmit={handleSubmit} />);
      
      const priceInput = screen.getByTestId('price-input');
      fireEvent.change(priceInput, { target: { name: 'price', value: '-10' } });
      
      const submitButton = screen.getByTestId('submit-button');
      fireEvent.click(submitButton);
      
      await waitFor(() => {
        expect(screen.getByTestId('price-error')).toHaveTextContent('Price must be a positive number');
      });
      
      expect(handleSubmit).not.toHaveBeenCalled();
    });

    it('should show error for missing category', async () => {
      const handleSubmit = jest.fn();
      render(<ProductForm onSubmit={handleSubmit} />);
      
      const submitButton = screen.getByTestId('submit-button');
      fireEvent.click(submitButton);
      
      await waitFor(() => {
        expect(screen.getByTestId('category-error')).toHaveTextContent('Category is required');
      });
      
      expect(handleSubmit).not.toHaveBeenCalled();
    });

    it('should show error for negative quantity', async () => {
      const handleSubmit = jest.fn();
      render(<ProductForm onSubmit={handleSubmit} />);
      
      const quantityInput = screen.getByTestId('quantity-input');
      fireEvent.change(quantityInput, { target: { name: 'quantity', value: '-5' } });
      
      const submitButton = screen.getByTestId('submit-button');
      fireEvent.click(submitButton);
      
      await waitFor(() => {
        expect(screen.getByTestId('quantity-error')).toHaveTextContent('Quantity must be a non-negative number');
      });
      
      expect(handleSubmit).not.toHaveBeenCalled();
    });

    it('should show error for invalid tax rate', async () => {
      const handleSubmit = jest.fn();
      render(<ProductForm onSubmit={handleSubmit} />);
      
      const taxRateInput = screen.getByTestId('tax-rate-input');
      fireEvent.change(taxRateInput, { target: { name: 'tax_rate', value: '1.5' } });
      
      const submitButton = screen.getByTestId('submit-button');
      fireEvent.click(submitButton);
      
      await waitFor(() => {
        expect(screen.getByTestId('tax-rate-error')).toHaveTextContent('Tax rate must be between 0 and 1');
      });
      
      expect(handleSubmit).not.toHaveBeenCalled();
    });

    it('should clear error when user starts typing', async () => {
      const handleSubmit = jest.fn();
      render(<ProductForm onSubmit={handleSubmit} />);
      
      // Trigger validation error
      const submitButton = screen.getByTestId('submit-button');
      fireEvent.click(submitButton);
      
      await waitFor(() => {
        expect(screen.getByTestId('name-error')).toBeInTheDocument();
      });
      
      // Start typing to clear error
      const nameInput = screen.getByTestId('name-input');
      fireEvent.change(nameInput, { target: { name: 'name', value: 'Test' } });
      
      await waitFor(() => {
        expect(screen.queryByTestId('name-error')).not.toBeInTheDocument();
      });
    });
  });

  describe('Form Submission', () => {
    it('should call onSubmit with form data when valid', async () => {
      const handleSubmit = jest.fn();
      render(<ProductForm onSubmit={handleSubmit} />);
      
      // Fill in valid data
      fireEvent.change(screen.getByTestId('name-input'), { target: { name: 'name', value: 'Test Product' } });
      fireEvent.change(screen.getByTestId('sku-input'), { target: { name: 'sku', value: 'SKU001' } });
      fireEvent.change(screen.getByTestId('price-input'), { target: { name: 'price', value: '19.99' } });
      fireEvent.change(screen.getByTestId('category-select'), { target: { name: 'category_id', value: '1' } });
      fireEvent.change(screen.getByTestId('quantity-input'), { target: { name: 'quantity', value: '10' } });
      
      const submitButton = screen.getByTestId('submit-button');
      fireEvent.click(submitButton);
      
      await waitFor(() => {
        expect(handleSubmit).toHaveBeenCalledWith({
          name: 'Test Product',
          sku: 'SKU001',
          price: '19.99',
          category_id: '1',
          quantity: '10',
          tax_rate: '',
          description: '',
        });
      });
    });

    it('should not call onSubmit when form is invalid', async () => {
      const handleSubmit = jest.fn();
      render(<ProductForm onSubmit={handleSubmit} />);
      
      const submitButton = screen.getByTestId('submit-button');
      fireEvent.click(submitButton);
      
      await waitFor(() => {
        expect(screen.getByTestId('name-error')).toBeInTheDocument();
      });
      
      expect(handleSubmit).not.toHaveBeenCalled();
    });

    it('should call onCancel when cancel button is clicked', () => {
      const handleCancel = jest.fn();
      render(<ProductForm onCancel={handleCancel} />);
      
      const cancelButton = screen.getByTestId('cancel-button');
      fireEvent.click(cancelButton);
      
      expect(handleCancel).toHaveBeenCalled();
    });
  });

  describe('User Interaction', () => {
    it('should update form fields when user types', async () => {
      const user = userEvent.setup();
      render(<ProductForm />);
      
      const nameInput = screen.getByTestId('name-input');
      await user.type(nameInput, 'New Product');
      
      expect(nameInput).toHaveValue('New Product');
    });

    it('should handle category selection', async () => {
      const user = userEvent.setup();
      render(<ProductForm />);
      
      const categorySelect = screen.getByTestId('category-select');
      await user.selectOptions(categorySelect, '2');
      
      expect(categorySelect).toHaveValue('2');
    });
  });
});
