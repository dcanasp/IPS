using Microsoft.AspNetCore.Authorization;
using Microsoft.AspNetCore.Mvc;

namespace victim.Controllers
{
    [ApiController]
    [Authorize(Roles = "Admin")]
    public class AdminController : Controller
    {
        [HttpGet("admin")]
        public IActionResult Get()
        {
            return Ok("This is the admin area. Only accessible by users with Admin role.");
        }
    }
}
