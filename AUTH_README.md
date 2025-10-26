# Authentication System Documentation

This document describes the new user authentication system implemented for the COMP 3007 webserver.

## Overview

The webserver has been upgraded from a single-password system to a full user authentication system using JSON Web Tokens (JWT). The system supports individual user accounts, role-based access control, and email-based account setup.

## Features

### 1. User Management
- Individual user accounts with email-based identification
- Password-based authentication using bcrypt hashing
- JWT tokens for session management
- SQLite database for user storage

### 2. Role-Based Access Control
- **User Role**: Can access course content and change their password
- **Admin Role**: Can manage users, add new users, and access admin features

### 3. Account Setup Process
- Administrators can add users by email address
- New users receive setup emails with secure tokens
- Users set their own passwords during initial setup
- Setup tokens expire after 7 days

### 4. Password Management
- Users can change their passwords after login
- Passwords must be at least 8 characters long
- Current password verification required for changes

### 5. Admin Features
- Bulk user creation from email lists
- User management dashboard with statistics
- Ability to resend setup emails to pending users
- View user status and creation dates

## Technical Implementation

### Database Schema
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

### JWT Claims Structure
```go
type AuthClaims struct {
    UserID  int    `json:"user_id"`
    Email   string `json:"email"`
    IsAdmin bool   `json:"is_admin"`
    jwt.RegisteredClaims
}
```

### Routes

#### Public Routes
- `GET/POST /login` - User authentication
- `GET/POST /setup?token=...` - Account setup with token
- `GET /logout` - User logout

#### Protected Routes (Requires Authentication)
- `GET /*` - All content pages (existing functionality)
- `GET/POST /change-password` - Password change form

#### Admin Routes (Requires Admin Role)
- `GET/POST /admin/add-users` - Add single or multiple users
- `GET /admin/manage-users` - User management dashboard
- `POST /admin/resend-setup-email` - Resend setup email

## Configuration

### Environment Variables

#### Email Configuration (Optional)
Set these environment variables to enable actual email sending:
```bash
SMTP_HOST=smtp.example.com
SMTP_PORT=587
SMTP_USER=username
SMTP_PASSWORD=password
FROM_EMAIL=noreply@comp3007.local
```

If email is not configured, setup links will be logged to the console for development.

#### JWT Secret (Production)
In production, set a secure JWT secret:
```bash
JWT_SECRET=your-very-secure-secret-key-here
```

### Database
- Default database file: `users.db` (SQLite)
- Created automatically on first run
- Default admin account: `admin@comp3007.local` with password `ahsahbeequen`

## Migration from Old System

The system automatically creates a default admin account with the old password (`ahsahbeequen`) for backward compatibility during migration. This account uses the email `admin@comp3007.local`.

## Usage Instructions

### For Administrators

1. **Login**: Use the default admin account or your assigned admin account
2. **Add Users**: Go to "Add Users" from the user menu
   - Add single users with email and optional admin role
   - Add multiple users by pasting email addresses (one per line)
3. **Manage Users**: View all users, their status, and resend setup emails
4. **Setup Emails**: When users are added, they automatically receive setup emails

### For Users

1. **Account Setup**: Click the link in your setup email
2. **Create Password**: Set a password (minimum 8 characters)
3. **Login**: Use your email and password to access the system
4. **Change Password**: Use the user menu to change your password

## Security Features

- Passwords are hashed using bcrypt with default cost
- JWT tokens expire after 24 hours
- Setup tokens expire after 7 days
- HttpOnly cookies prevent XSS attacks
- CSRF protection through SameSite cookie policy
- Input validation and sanitization

## UI Components

All new pages follow the existing design system:
- Consistent styling with Tailwind CSS
- Responsive mobile-friendly layouts
- Form validation and error handling
- Loading states and user feedback
- Accessible form controls and navigation

## Development

### Building
```bash
go build -o server .
```

### Running
```bash
./server
```

The server will:
1. Create the SQLite database if it doesn't exist
2. Set up the default admin account
3. Start listening on port 8080 (or PORT environment variable)

### Testing Email
During development, setup emails are logged to the console. Copy the setup URL from the logs to test account creation.

## Troubleshooting

### Common Issues

1. **Database locked**: Ensure no other instance is running
2. **Email not sending**: Check SMTP configuration and logs
3. **Setup token expired**: Admin can resend setup emails
4. **Forgotten password**: Admins must create a new account or reset via database

### Logs
All authentication events are logged with appropriate detail levels for debugging and security monitoring.
