# Authentication System Changes

This document summarizes all the changes made to convert the COMP 3007 webserver from a single-password system to a comprehensive user authentication system with JSON Web Tokens.

## Overview

The webserver has been completely upgraded with:
- Individual user accounts with email-based authentication
- JWT tokens for secure session management
- Role-based access control (User and Admin roles)
- Email-based account setup process
- Password management capabilities
- Admin interface for user management

## Files Added

### Core Authentication System
- `auth.go` - Complete authentication manager with JWT, user management, and email functionality
- `users.db` - SQLite database (created automatically on first run)

### Templates
- `templates/user-login.html` - New login page with email/password fields
- `templates/setup.html` - Account setup page for new users
- `templates/change-password.html` - Password change form for users
- `templates/admin-add-users.html` - Admin interface for adding users (single and bulk)
- `templates/admin-manage-users.html` - Admin dashboard for user management

### Documentation
- `AUTH_README.md` - Comprehensive documentation of the authentication system
- `CHANGES.md` - This file, documenting all modifications

### Sample Site Content
- `site/index.md` - Updated homepage with authentication info
- `site/outline.md` - Course outline
- `site/assignments.md` - Assignment descriptions
- `site/quizzes.md` - Quiz information
- `site/lectures.md` - Lecture materials
- `site/reference.md` - Programming language references

## Files Modified

### `main.go`
- **Complete rewrite** of authentication system
- Removed single password authentication
- Added JWT-based user authentication
- Added new route handlers for:
  - User login (`/login`)
  - Account setup (`/setup`)
  - Password changes (`/change-password`)
  - User logout (`/logout`)
  - Admin user management (`/admin/add-users`, `/admin/manage-users`)
  - Setup email resending (`/admin/resend-setup-email`)
- Updated page serving to include user context
- Added middleware for authentication and admin authorization

### `templates/navigation.html`
- Added user menu with dropdown
- Added admin-specific navigation items
- Added logout functionality
- Mobile-responsive user menu

### `templates/login.html`
- Renamed to provide fallback, but `user-login.html` is the primary template

### `go.mod`
- Added JWT library: `github.com/golang-jwt/jwt/v5`
- Added SQLite driver: `github.com/mattn/go-sqlite3`
- Added email library: `gopkg.in/gomail.v2`

## Database Schema

### Users Table
```sql
CREATE TABLE users (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    email TEXT UNIQUE NOT NULL,
    password TEXT,
    is_admin BOOLEAN DEFAULT FALSE,
    setup_token TEXT,
    setup_token_expiry DATETIME,
    is_setup BOOLEAN DEFAULT FALSE,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);
```

## Key Features Implemented

### 1. User Authentication
- Email-based user identification
- Secure password hashing with bcrypt
- JWT tokens for session management (24-hour expiry)
- Automatic redirect to login for unauthenticated users

### 2. Account Setup Process
- Administrators can create user accounts by email
- New users receive setup emails with secure tokens
- Users set their own passwords during initial setup
- Setup tokens expire after 7 days
- Ability to resend setup emails for pending users

### 3. Password Management
- Users can change their passwords after login
- Current password verification required
- Minimum 8-character password requirement
- Password confirmation validation

### 4. Role-Based Access Control
- **User Role**: Access to course content and password change
- **Admin Role**: All user capabilities plus:
  - Add new users (single or bulk)
  - View all users and their status
  - Resend setup emails
  - Access admin dashboard

### 5. Admin Features
- **User Management Dashboard**:
  - Statistics: total users, setup complete, pending setup
  - User list with status, role, and creation date
  - Search and filter functionality
  - User details modal
- **Bulk User Creation**:
  - Add single users with email and admin flag
  - Bulk import from email list (one per line)
  - Email validation and duplicate detection
  - Success/error reporting

### 6. Email System
- Configurable SMTP settings via environment variables
- Development mode: emails logged to console
- Production mode: actual email sending via SMTP
- HTML email templates with setup links

## Security Features

- **Password Security**: bcrypt hashing with default cost
- **Token Security**: JWT tokens with HMAC-SHA256 signing
- **Cookie Security**: HttpOnly, SameSite=Lax cookies
- **Input Validation**: Email format validation, password length requirements
- **SQL Injection Protection**: Parameterized queries
- **XSS Prevention**: Template escaping, HttpOnly cookies

## Migration Path

### From Old System
1. Default admin account created automatically: `admin@comp3007.local` / `ahsahbeequen`
2. Existing users need new accounts created by admin
3. No data loss - all course content remains accessible

### Environment Configuration

#### Required (None - works out of the box)
- Database created automatically
- Default admin account created on first run

#### Optional Email Configuration
```bash
SMTP_HOST=smtp.example.com
SMTP_PORT=587
SMTP_USER=username
SMTP_PASSWORD=password
FROM_EMAIL=noreply@comp3007.local
```

#### Optional JWT Secret (Production)
```bash
JWT_SECRET=your-very-secure-secret-key-here
```

## User Experience

### For Students
1. Receive setup email from instructor
2. Click setup link to create password
3. Login with email and password
4. Access all course materials
5. Change password as needed

### For Instructors/Admins
1. Login with admin account
2. Add students via "Add Users" page:
   - Single user: enter email, check admin if needed
   - Bulk users: paste email list, validate, submit
3. Monitor user status via "Manage Users" page
4. Resend setup emails for users who haven't completed setup
5. Manage own password via user menu

## Testing

### Manual Testing Checklist
- [ ] Server starts without errors
- [ ] Default admin account works
- [ ] User creation (single and bulk)
- [ ] Setup email generation (check console logs)
- [ ] Account setup with valid token
- [ ] Login with created account
- [ ] Password change functionality
- [ ] Admin vs user access control
- [ ] Logout functionality
- [ ] Session timeout handling

### Database Testing
- [ ] Users table created correctly
- [ ] Default admin inserted
- [ ] User creation/updates work
- [ ] Token generation and expiry
- [ ] Password hashing verification

## Performance Considerations

- **Database**: SQLite suitable for course-sized user base (hundreds of users)
- **Memory**: JWT tokens stored only in cookies, minimal server memory usage
- **Scalability**: Can be upgraded to PostgreSQL/MySQL for larger deployments
- **Caching**: No special caching needed for typical course workload

## Deployment Notes

### Development
```bash
go build -o server .
./server
```

### Production Recommendations
- Set JWT_SECRET environment variable
- Configure SMTP for email sending
- Use HTTPS (set Secure flag on cookies)
- Regular database backups
- Monitor logs for security events

## Backward Compatibility

- All existing course content and navigation preserved
- Same URL structure for content pages
- Same styling and responsive design
- File upload functionality preserved with authentication

## Future Enhancements

Possible future improvements:
- Password reset functionality via email
- Two-factor authentication
- User profile management
- Audit logging for admin actions
- Bulk user import from CSV files
- Integration with university LDAP/SSO
- API endpoints for external integrations

## Support

For issues or questions:
1. Check AUTH_README.md for detailed documentation
2. Review server logs for error messages
3. Verify environment configuration
4. Test with default admin account
5. Contact system administrator for database issues

---

*All changes maintain the existing course website functionality while adding comprehensive user management capabilities.*
