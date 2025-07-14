using Microsoft.AspNetCore.Mvc;
using victim.Persistence;
namespace victim.Controllers
{
    [ApiController]
    public class DataController : ControllerBase
    {
        [HttpGet("/search")]
        public IActionResult Search([FromQuery] string q)
        {
            var results = ProductStore.Products
                .Where(p => p.Name.Contains(q, StringComparison.OrdinalIgnoreCase) ||
                            p.Description.Contains(q, StringComparison.OrdinalIgnoreCase))
                .ToList();

            return Ok(results);
        }

        [HttpGet("/api/data/{id}")]
        public IActionResult GetItem(int id)
        {
            var product = ProductStore.Products.FirstOrDefault(p => p.Id == id);
            if (product == null)
                return NotFound("Item not found");

            return Ok(product);
        }
    }
}
