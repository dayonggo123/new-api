package service

import (
	"testing"
)

func TestExtractTikHubProductV3Info(t *testing.T) {
	body := []byte(`{
		"code": 200,
		"data": {
			"product_data": {
				"page_config": {
					"components_map": [
						{
							"component_type": "product_info",
							"component_data": {
								"product_info": {
									"product_info": {
										"product_model": {
											"product_id": "1729556031006938037",
											"name": "Test Product",
											"description": [
												{"type": "text", "sub": [{"t": "Line one."}]},
												{"type": "text", "sub": [{"t": "Line two."}]}
											],
											"images": [
												{"url_list": ["https://img1.example.com/a.jpg"]},
												{"url_list": ["https://img2.example.com/b.jpg"]}
											],
											"sold_count": 100,
											"sale_properties": [
												{
													"property_id": "p1",
													"property_name": "Flavor",
													"has_image": true,
													"property_values": [
														{"property_value_id": "v1", "property_value_name": "Mango", "image": {"url_list": ["https://img-mango.example.com"]}}
													]
												}
											],
											"product_properties": [
												{
													"property_id": "pp1",
													"property_name": "Sugar Free",
													"property_values": [
														{"property_value_id": "pv1", "property_value_name": "Yes"}
													]
												}
											],
											"skus": [
												{
													"sku_name": "MANGO1",
													"sku_status": 1,
													"sku_stock_status": 1,
													"package_weight": 10,
													"package_length": 6,
													"package_width": 11,
													"package_height": 16,
													"sku_quantity": {"available_quantity": 500},
													"property_pairs": [
														{"sku_property_name": "Flavor", "sku_property_value_name": "Mango"}
													]
												}
											]
										},
										"review_model": {
											"product_overall_score": 4.5,
											"product_review_count": "1234"
										},
										"seller_model": {
											"shop_name": "Test Shop",
											"shop_logo": {"url_list": ["https://logo.example.com"]}
										}
									}
								}
							}
						}
					]
				}
			}
		}
	}`)

	info, err := ExtractTikHubProductV3Info(body)
	if err != nil {
		t.Fatalf("extract failed: %v", err)
	}

	if info.ProductID != "1729556031006938037" {
		t.Errorf("product_id = %s, want 1729556031006938037", info.ProductID)
	}
	if info.Title != "Test Product" {
		t.Errorf("title = %s, want Test Product", info.Title)
	}
	if info.SoldCount != 100 {
		t.Errorf("sold_count = %d, want 100", info.SoldCount)
	}
	if info.Rating != 4.5 {
		t.Errorf("rating = %f, want 4.5", info.Rating)
	}
	if info.ReviewCount != 1234 {
		t.Errorf("review_count = %d, want 1234", info.ReviewCount)
	}
	if info.ShopName != "Test Shop" {
		t.Errorf("shop_name = %s, want Test Shop", info.ShopName)
	}
	if len(info.Images) != 2 {
		t.Errorf("images len = %d, want 2", len(info.Images))
	}
	if len(info.SaleProperties) != 1 {
		t.Errorf("sale_properties len = %d, want 1", len(info.SaleProperties))
	}
	if len(info.ProductProperties) != 1 {
		t.Errorf("product_properties len = %d, want 1", len(info.ProductProperties))
	}
	if len(info.SKUs) != 1 {
		t.Errorf("skus len = %d, want 1", len(info.SKUs))
	}
	if info.Package.Weight != 10 || info.Package.Length != 6 {
		t.Errorf("package = %+v, want weight=10 length=6", info.Package)
	}
	if info.SKUs[0].Properties["Flavor"] != "Mango" {
		t.Errorf("sku flavor = %s, want Mango", info.SKUs[0].Properties["Flavor"])
	}
}
