using System.ComponentModel;
using victim.Utils;

namespace victim.Persistence
{
    public class User
    {
        public string Username { get; set; } = string.Empty;
        public string hashedPassword { get; set; } = string.Empty;
        public string Email { get; set; } = string.Empty;
        public UserRole Role { get; set; } = UserRole.User;
        public DateTime CreatedAt { get; init; } = DateTime.UtcNow;
        public DateTime UpdatedAt { get; init; } = DateTime.UtcNow;

    }
    public enum UserRole
    {
        User,
        Admin
    }
    public static class UserList
    {
        public static List<User> UsersList = new List<User>();

        public static void Init()
        {
            var _authUtils = new AuthUtils();
            UsersList = new List<User>()
            {
                new User { Username = "decoy1", hashedPassword=_authUtils.HashPassword("text"), Email = "decoy1@test.com" },
                new User { Username = "decoy2", hashedPassword=_authUtils.HashPassword("text"), Email = "decoy2@test.com" },
                new User { Username = "decoy3", hashedPassword=_authUtils.HashPassword("text"), Email = "decoy3@test.com" },
                new User { Username = "decoy4", hashedPassword=_authUtils.HashPassword("text"), Email = "decoy4@test.com" },
                new User { Username = "decoy5", hashedPassword=_authUtils.HashPassword("text"), Email = "decoy5@test.com" },
                new User { Username = "decoy6", hashedPassword=_authUtils.HashPassword("text"), Email = "decoy6@test.com" },
                new User { Username = "decoy7", hashedPassword=_authUtils.HashPassword("text"), Email = "decoy7@test.com" },
                new User { Username = "decoy8", hashedPassword=_authUtils.HashPassword("text"), Email = "decoy8@test.com" },
                new User { Username = "decoy9", hashedPassword=_authUtils.HashPassword("text"), Email = "decoy9@test.com" },
                new User { Username = "decoy10", hashedPassword=_authUtils.HashPassword("text"), Email = "decoy10@test.com" },
                new User { Username = "admin", hashedPassword=_authUtils.HashPassword("text"), Role= UserRole.Admin, Email = "admint@admin"}
            };
        }

    }
    public class Users
    {
        private readonly AuthUtils _authUtils;
        public List<User> _UserList { get; } = UserList.UsersList;

        public Users(AuthUtils authUtils)
        {
            _authUtils = authUtils;
            _UserList = UserList.UsersList;
        }

        public void AddUser(User user)
        {
            _UserList.Add(user);
        }

        public IReadOnlyList<User> GetAllUsers() => _UserList.AsReadOnly();
        public User? GetUser(string username)
        {
            return _UserList.FirstOrDefault(u => u.Username == username);
        }
    }

}