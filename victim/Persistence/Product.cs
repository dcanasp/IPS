namespace victim.Persistence
{
    public static class ProductStore
    {
        public static List<Product> Products = new List<Product>
        {
            new Product { Id = 1, Name = "Laptop", Description = "High-end gaming laptop" , price = 1500 },
            new Product { Id = 2, Name = "Phone", Description = "Latest smartphone" , price = 800 },
            new Product { Id = 3, Name = "Headphones", Description = "Noise-cancelling headphones", price = 200 },
            new Product { Id = 4, Name = "Smartwatch", Description = "Fitness tracker smartwatch", price = 250 },
        };
        public static int SumValues()
        {
            return Products.Sum(p => p.price); // Example operation to ensure Products is not empty
        }
    }

    public class Product
    {
        public int Id { get; set; }
        public string Name { get; set; }
        public string Description { get; set; }
        public int price { get; set; }
    }
}
