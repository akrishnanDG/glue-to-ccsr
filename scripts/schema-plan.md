# Schema Registration Plan for AWS Glue SR

## Registry Details
- **Registry Name:** payments-regsitry
- **ARN:** arn:aws:glue:us-east-2:829250931565:registry/payments-regsitry
- **Region:** us-east-2

## Compatibility Strategy
- **Backward Compatible (30 subjects):** Can read old data with new schema
- **Forward Compatible (20 subjects):** Can read new data with old schema

## 50 Schema Subjects (Avoiding "payment" keyword)

### E-commerce Domain (15 subjects) - BACKWARD Compatible
1. **order-placed** - Order creation events (3 versions)
2. **order-shipped** - Shipping notifications (4 versions)
3. **order-delivered** - Delivery confirmations (3 versions)
4. **product-created** - Product catalog entries (5 versions)
5. **product-updated** - Product modifications (4 versions)
6. **inventory-adjusted** - Stock level changes (3 versions)
7. **cart-item-added** - Shopping cart additions (3 versions)
8. **cart-item-removed** - Cart item removals (3 versions)
9. **checkout-initiated** - Checkout start events (4 versions)
10. **checkout-completed** - Checkout completions (3 versions)
11. **refund-requested** - Refund requests (4 versions)
12. **refund-processed** - Refund completions (3 versions)
13. **shipment-tracking** - Tracking updates (5 versions)
14. **delivery-exception** - Delivery issues (3 versions)
15. **return-initiated** - Product returns (4 versions)

### Customer Domain (10 subjects) - BACKWARD Compatible
16. **customer-registered** - New customer signups (4 versions)
17. **customer-profile-updated** - Profile changes (5 versions)
18. **customer-address-changed** - Address updates (3 versions)
19. **customer-preference-set** - Preference updates (3 versions)
20. **customer-verified** - Email/phone verification (3 versions)
21. **customer-deactivated** - Account deactivation (3 versions)
22. **loyalty-points-earned** - Rewards earned (4 versions)
23. **loyalty-points-redeemed** - Rewards used (3 versions)
24. **customer-review-posted** - Product reviews (4 versions)
25. **customer-complaint-filed** - Support tickets (4 versions)

### Transaction Domain (10 subjects) - FORWARD Compatible
26. **transaction-authorized** - Auth events (4 versions)
27. **transaction-captured** - Capture events (4 versions)
28. **transaction-voided** - Void events (3 versions)
29. **transaction-refunded** - Refund events (4 versions)
30. **transaction-declined** - Decline events (3 versions)
31. **invoice-generated** - Invoice creation (5 versions)
32. **invoice-sent** - Invoice delivery (3 versions)
33. **invoice-paid** - Payment received (4 versions)
34. **subscription-created** - Subscription start (4 versions)
35. **subscription-renewed** - Renewal events (3 versions)

### Analytics Domain (5 subjects) - FORWARD Compatible
36. **page-view-tracked** - Web analytics (5 versions)
37. **button-click-tracked** - Click events (4 versions)
38. **search-performed** - Search queries (4 versions)
39. **recommendation-shown** - Recommendations (3 versions)
40. **recommendation-clicked** - Recommendation clicks (3 versions)

### Marketing Domain (5 subjects) - FORWARD Compatible
41. **email-campaign-sent** - Email sends (4 versions)
42. **email-opened** - Email opens (3 versions)
43. **email-clicked** - Email clicks (4 versions)
44. **promotion-applied** - Promo code usage (4 versions)
45. **discount-applied** - Discount events (3 versions)

### Inventory Domain (5 subjects) - BACKWARD Compatible
46. **warehouse-transfer** - Stock transfers (4 versions)
47. **stock-count-adjusted** - Inventory counts (3 versions)
48. **supplier-order-placed** - Purchase orders (5 versions)
49. **supplier-order-received** - PO receipts (4 versions)
50. **low-stock-alert** - Stock warnings (3 versions)

## Schema Evolution Strategy

### Version Progression Pattern:
- **v1:** Basic required fields only
- **v2:** Add optional metadata fields (backward compatible)
- **v3:** Add nested objects/arrays (backward compatible)
- **v4:** Add computed fields or IDs (backward compatible)
- **v5:** Add audit/compliance fields (backward compatible)

### Backward Compatible Changes:
- Adding optional fields (with defaults)
- Adding enum values
- Removing required constraints

### Forward Compatible Changes:
- Removing optional fields
- Making required fields optional
- Changing field types to unions
