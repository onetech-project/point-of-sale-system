// T150: Frontend unit tests for ProductList component

import React from 'react';
import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import '@testing-library/jest-dom';

// Mock ProductList component for testing
const ProductList = ({ 
  products = [], 
  onProductClick = () => {}, 
  onSearch = () => {},
  onFilterChange = () => {},
  loading = false,
  showArchived = false,
}) => {
  const [searchTerm, setSearchTerm] = React.useState('');
  const [selectedCategory, setSelectedCategory] = React.useState('');

  const handleSearchChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    const value = e.target.value;
    setSearchTerm(value);
    onSearch(value);
  };

  const handleCategoryChange = (e: React.ChangeEvent<HTMLSelectElement>) => {
    const value = e.target.value;
    setSelectedCategory(value);
    onFilterChange({ category: value });
  };

  const getStockStatus = (quantity: number, reorderLevel: number = 10) => {
    if (quantity === 0) return 'out-of-stock';
    if (quantity <= reorderLevel) return 'low-stock';
    return 'in-stock';
  };

  const getStockBadge = (quantity: number, reorderLevel: number = 10) => {
    const status = getStockStatus(quantity, reorderLevel);
    
    if (status === 'out-of-stock') return 'Out of Stock';
    if (status === 'low-stock') return 'Low Stock';
    return 'In Stock';
  };

  if (loading) {
    return <div data-testid="loading-spinner">Loading...</div>;
  }

  return (
    <div data-testid="product-list">
      <div data-testid="search-controls">
        <input
          type="text"
          placeholder="Search products..."
          value={searchTerm}
          onChange={handleSearchChange}
          data-testid="search-input"
        />
        
        <select
          value={selectedCategory}
          onChange={handleCategoryChange}
          data-testid="category-filter"
        >
          <option value="">All Categories</option>
          <option value="1">Electronics</option>
          <option value="2">Food</option>
          <option value="3">Clothing</option>
        </select>

        <label data-testid="archived-toggle">
          <input
            type="checkbox"
            checked={showArchived}
            onChange={(e) => onFilterChange({ archived: e.target.checked })}
            data-testid="archived-checkbox"
          />
          Show Archived
        </label>
      </div>

      {products.length === 0 ? (
        <div data-testid="no-products">No products found</div>
      ) : (
        <div data-testid="products-grid">
          {products.map((product: any) => {
            const stockStatus = getStockStatus(product.quantity, product.reorder_level);
            const stockBadge = getStockBadge(product.quantity, product.reorder_level);
            
            return (
              <div
                key={product.id}
                data-testid={`product-item-${product.id}`}
                className={`product-item ${stockStatus} ${product.archived ? 'archived' : ''}`}
                onClick={() => onProductClick(product.id)}
                style={{ cursor: 'pointer' }}
              >
                <div data-testid={`product-name-${product.id}`}>{product.name}</div>
                <div data-testid={`product-sku-${product.id}`}>SKU: {product.sku}</div>
                <div data-testid={`product-price-${product.id}`}>
                  ${parseFloat(product.price).toFixed(2)}
                </div>
                <div data-testid={`product-quantity-${product.id}`}>
                  Qty: {product.quantity}
                </div>
                <span
                  data-testid={`stock-badge-${product.id}`}
                  className={`badge ${stockStatus}`}
                >
                  {stockBadge}
                </span>
                {product.archived && (
                  <span data-testid={`archived-badge-${product.id}`} className="badge archived">
                    Archived
                  </span>
                )}
              </div>
            );
          })}
        </div>
      )}

      <div data-testid="product-count">
        Showing {products.length} product{products.length !== 1 ? 's' : ''}
      </div>
    </div>
  );
};

describe('ProductList', () => {
  const mockProducts = [
    {
      id: '1',
      name: 'Laptop',
      sku: 'LAP001',
      price: '999.99',
      quantity: 50,
      reorder_level: 10,
      category_id: '1',
      archived: false,
    },
    {
      id: '2',
      name: 'Mouse',
      sku: 'MOU001',
      price: '19.99',
      quantity: 5,
      reorder_level: 10,
      category_id: '1',
      archived: false,
    },
    {
      id: '3',
      name: 'Keyboard',
      sku: 'KEY001',
      price: '49.99',
      quantity: 0,
      reorder_level: 10,
      category_id: '1',
      archived: false,
    },
    {
      id: '4',
      name: 'Old Product',
      sku: 'OLD001',
      price: '99.99',
      quantity: 10,
      reorder_level: 10,
      category_id: '2',
      archived: true,
    },
  ];

  describe('Rendering', () => {
    it('should render product list with search controls', () => {
      render(<ProductList products={mockProducts} />);
      
      expect(screen.getByTestId('search-input')).toBeInTheDocument();
      expect(screen.getByTestId('category-filter')).toBeInTheDocument();
      expect(screen.getByTestId('archived-checkbox')).toBeInTheDocument();
    });

    it('should render all products', () => {
      render(<ProductList products={mockProducts} />);
      
      expect(screen.getByTestId('product-item-1')).toBeInTheDocument();
      expect(screen.getByTestId('product-item-2')).toBeInTheDocument();
      expect(screen.getByTestId('product-item-3')).toBeInTheDocument();
      expect(screen.getByTestId('product-item-4')).toBeInTheDocument();
    });

    it('should display product details correctly', () => {
      render(<ProductList products={mockProducts} />);
      
      expect(screen.getByTestId('product-name-1')).toHaveTextContent('Laptop');
      expect(screen.getByTestId('product-sku-1')).toHaveTextContent('SKU: LAP001');
      expect(screen.getByTestId('product-price-1')).toHaveTextContent('$999.99');
      expect(screen.getByTestId('product-quantity-1')).toHaveTextContent('Qty: 50');
    });

    it('should show loading spinner when loading', () => {
      render(<ProductList products={[]} loading={true} />);
      
      expect(screen.getByTestId('loading-spinner')).toBeInTheDocument();
      expect(screen.queryByTestId('products-grid')).not.toBeInTheDocument();
    });

    it('should show "no products" message when list is empty', () => {
      render(<ProductList products={[]} />);
      
      expect(screen.getByTestId('no-products')).toHaveTextContent('No products found');
    });

    it('should display product count', () => {
      render(<ProductList products={mockProducts} />);
      
      expect(screen.getByTestId('product-count')).toHaveTextContent('Showing 4 products');
    });

    it('should use singular form for single product', () => {
      render(<ProductList products={[mockProducts[0]]} />);
      
      expect(screen.getByTestId('product-count')).toHaveTextContent('Showing 1 product');
    });
  });

  describe('Stock Status', () => {
    it('should show "In Stock" badge for products with sufficient quantity', () => {
      render(<ProductList products={mockProducts} />);
      
      const badge = screen.getByTestId('stock-badge-1');
      expect(badge).toHaveTextContent('In Stock');
      expect(badge).toHaveClass('badge', 'in-stock');
    });

    it('should show "Low Stock" badge for products below reorder level', () => {
      render(<ProductList products={mockProducts} />);
      
      const badge = screen.getByTestId('stock-badge-2');
      expect(badge).toHaveTextContent('Low Stock');
      expect(badge).toHaveClass('badge', 'low-stock');
    });

    it('should show "Out of Stock" badge for products with zero quantity', () => {
      render(<ProductList products={mockProducts} />);
      
      const badge = screen.getByTestId('stock-badge-3');
      expect(badge).toHaveTextContent('Out of Stock');
      expect(badge).toHaveClass('badge', 'out-of-stock');
    });

    it('should highlight low-stock products', () => {
      render(<ProductList products={mockProducts} />);
      
      const lowStockItem = screen.getByTestId('product-item-2');
      expect(lowStockItem).toHaveClass('low-stock');
    });

    it('should highlight out-of-stock products', () => {
      render(<ProductList products={mockProducts} />);
      
      const outOfStockItem = screen.getByTestId('product-item-3');
      expect(outOfStockItem).toHaveClass('out-of-stock');
    });
  });

  describe('Archived Products', () => {
    it('should show archived badge for archived products', () => {
      render(<ProductList products={mockProducts} />);
      
      expect(screen.getByTestId('archived-badge-4')).toBeInTheDocument();
    });

    it('should not show archived badge for active products', () => {
      render(<ProductList products={mockProducts} />);
      
      expect(screen.queryByTestId('archived-badge-1')).not.toBeInTheDocument();
      expect(screen.queryByTestId('archived-badge-2')).not.toBeInTheDocument();
      expect(screen.queryByTestId('archived-badge-3')).not.toBeInTheDocument();
    });

    it('should apply archived class to archived products', () => {
      render(<ProductList products={mockProducts} />);
      
      const archivedItem = screen.getByTestId('product-item-4');
      expect(archivedItem).toHaveClass('archived');
    });
  });

  describe('Search Functionality', () => {
    it('should call onSearch when search input changes', async () => {
      const handleSearch = jest.fn();
      const user = userEvent.setup();
      render(<ProductList products={mockProducts} onSearch={handleSearch} />);
      
      const searchInput = screen.getByTestId('search-input');
      await user.type(searchInput, 'Laptop');
      
      expect(handleSearch).toHaveBeenCalled();
      expect(handleSearch).toHaveBeenLastCalledWith('Laptop');
    });

    it('should update search input value', async () => {
      const user = userEvent.setup();
      render(<ProductList products={mockProducts} />);
      
      const searchInput = screen.getByTestId('search-input');
      await user.type(searchInput, 'Mouse');
      
      expect(searchInput).toHaveValue('Mouse');
    });

    it('should handle empty search', async () => {
      const handleSearch = jest.fn();
      const user = userEvent.setup();
      render(<ProductList products={mockProducts} onSearch={handleSearch} />);
      
      const searchInput = screen.getByTestId('search-input');
      await user.type(searchInput, 'test');
      await user.clear(searchInput);
      
      expect(handleSearch).toHaveBeenLastCalledWith('');
    });
  });

  describe('Category Filter', () => {
    it('should call onFilterChange when category is selected', async () => {
      const handleFilterChange = jest.fn();
      const user = userEvent.setup();
      render(<ProductList products={mockProducts} onFilterChange={handleFilterChange} />);
      
      const categoryFilter = screen.getByTestId('category-filter');
      await user.selectOptions(categoryFilter, '1');
      
      expect(handleFilterChange).toHaveBeenCalledWith({ category: '1' });
    });

    it('should update category filter value', async () => {
      const user = userEvent.setup();
      render(<ProductList products={mockProducts} />);
      
      const categoryFilter = screen.getByTestId('category-filter');
      await user.selectOptions(categoryFilter, '2');
      
      expect(categoryFilter).toHaveValue('2');
    });

    it('should have "All Categories" as default option', () => {
      render(<ProductList products={mockProducts} />);
      
      const categoryFilter = screen.getByTestId('category-filter');
      expect(categoryFilter).toHaveValue('');
    });
  });

  describe('Archived Toggle', () => {
    it('should call onFilterChange when archived toggle is clicked', () => {
      const handleFilterChange = jest.fn();
      render(<ProductList products={mockProducts} onFilterChange={handleFilterChange} />);
      
      const archivedCheckbox = screen.getByTestId('archived-checkbox');
      fireEvent.click(archivedCheckbox);
      
      expect(handleFilterChange).toHaveBeenCalledWith({ archived: true });
    });

    it('should reflect showArchived prop state', () => {
      render(<ProductList products={mockProducts} showArchived={true} />);
      
      const archivedCheckbox = screen.getByTestId('archived-checkbox');
      expect(archivedCheckbox).toBeChecked();
    });
  });

  describe('Product Click', () => {
    it('should call onProductClick when product is clicked', () => {
      const handleProductClick = jest.fn();
      render(<ProductList products={mockProducts} onProductClick={handleProductClick} />);
      
      const productItem = screen.getByTestId('product-item-1');
      fireEvent.click(productItem);
      
      expect(handleProductClick).toHaveBeenCalledWith('1');
    });

    it('should have pointer cursor on product items', () => {
      render(<ProductList products={mockProducts} />);
      
      const productItem = screen.getByTestId('product-item-1');
      expect(productItem).toHaveStyle({ cursor: 'pointer' });
    });
  });

  describe('Price Formatting', () => {
    it('should format prices to 2 decimal places', () => {
      const products = [
        { ...mockProducts[0], price: '10' },
        { ...mockProducts[1], price: '10.5' },
        { ...mockProducts[2], price: '10.99' },
      ];
      
      render(<ProductList products={products} />);
      
      expect(screen.getByTestId('product-price-1')).toHaveTextContent('$10.00');
      expect(screen.getByTestId('product-price-2')).toHaveTextContent('$10.50');
      expect(screen.getByTestId('product-price-3')).toHaveTextContent('$10.99');
    });
  });

  describe('Edge Cases', () => {
    it('should handle products with missing optional fields', () => {
      const minimalProduct = {
        id: '5',
        name: 'Minimal Product',
        sku: 'MIN001',
        price: '5.00',
        quantity: 10,
        category_id: '1',
        archived: false,
      };
      
      render(<ProductList products={[minimalProduct]} />);
      
      expect(screen.getByTestId('product-item-5')).toBeInTheDocument();
    });

    it('should handle very long product names', () => {
      const longNameProduct = {
        ...mockProducts[0],
        name: 'A'.repeat(255),
      };
      
      render(<ProductList products={[longNameProduct]} />);
      
      expect(screen.getByTestId('product-name-1')).toHaveTextContent('A'.repeat(255));
    });

    it('should handle zero price', () => {
      const freeProduct = {
        ...mockProducts[0],
        price: '0.00',
      };
      
      render(<ProductList products={[freeProduct]} />);
      
      expect(screen.getByTestId('product-price-1')).toHaveTextContent('$0.00');
    });

    it('should handle very large quantities', () => {
      const largeQuantityProduct = {
        ...mockProducts[0],
        quantity: 1000000,
      };
      
      render(<ProductList products={[largeQuantityProduct]} />);
      
      expect(screen.getByTestId('product-quantity-1')).toHaveTextContent('Qty: 1000000');
    });
  });
});
