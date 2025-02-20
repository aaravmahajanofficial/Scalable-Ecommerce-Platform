package repository

import (
	"database/sql"
	"errors"

	"github.com/aaravmahajanofficial/scalable-ecommerce-platform/internal/models"
)

type ProductRepository struct {
	DB *sql.DB
}

func NewProductRepo(db *sql.DB) *ProductRepository {
	return &ProductRepository{DB: db}
}

func (p *ProductRepository) CreateProduct(product *models.Product) error {

	query := `INSERT INTO products (category_id, name, description, price, stock_quantity, sku, status)
			  VALUES ($1, $2, $3, $4, $5, $6, $7)
			  RETURNING id, created_at, updated_at
	`

	return p.DB.QueryRow(query, product.CategoryID, product.Name, product.Description, product.Price, product.StockQuantity, product.SKU, "active").Scan(&product.ID, &product.CreatedAt, &product.UpdatedAt)

}

func (p *ProductRepository) GetProductByID(id int64) (*models.Product, error) {

	product := &models.Product{}

	query := `
        SELECT p.id, p.category_id, p.name, p.description, p.price, 
               p.stock_quantity, p.sku, p.status, p.created_at, p.updated_at,
               c.id, c.name, c.description
        FROM products p
        LEFT JOIN categories c ON p.category_id = c.id
        WHERE p.id = $1`

	var category models.Category

	err := p.DB.QueryRow(query, id).Scan(&product.ID, &product.CategoryID, &product.Name, &product.Description, &product.Price, &product.StockQuantity, &product.SKU, &product.Status, &product.CreatedAt, &product.UpdatedAt, &category.ID, &category.Name, &category.Description)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, errors.New("product not found")
		}

		return nil, err
	}

	product.Category = &category
	return product, nil
}

func (p *ProductRepository) UpdateProduct(product *models.Product) error {

	query := `
		UPDATE products SET category_id = $1, name = $2, description = $3, price = $4, stock_quantity = $5, status = $6, updated_at = NOW()
		WHERE id = $7
		RETURNING updated_at
	`

	return p.DB.QueryRow(query, product.CategoryID, product.Name, product.Description, product.Price, product.StockQuantity, product.Status, product.ID).Scan(&product.UpdatedAt)

}

func (p *ProductRepository) ListProducts(offset, limit int) ([]*models.Product, error) {

	query := `
		SELECT p.id, p.category_id, p.name, p.description, p.price, 
		p.stock_quantity, p.sku, p.status, p.created_at, p.updated_at,
		c.id, c.name, c.description
		FROM products p
		LEFT JOIN categories c on p.category_id = c.id
		ORDER BY p.id
		LIMIT $1 OFFSET $2
	`

	rows, err := p.DB.Query(query, limit, offset)

	if err != nil {
		return nil, err
	}

	defer rows.Close()

	var products []*models.Product

	for rows.Next() {

		product := &models.Product{}
		category := &models.Category{}

		err := rows.Scan(&product.ID, &product.CategoryID, &product.Name, &product.Description, &product.Price, &product.StockQuantity, &product.SKU, &product.Status, &product.CreatedAt, &product.UpdatedAt, &category.ID, &category.Name, &category.Description)

		if err != nil {
			return nil, err
		}

		product.Category = category
		products = append(products, product)

	}

	return products, nil
}
