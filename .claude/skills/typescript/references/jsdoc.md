# JSDoc Documentation

TypeDoc-compatible documentation standards for TypeScript code.

## Table of Contents

- [Function Documentation](#function-documentation)
- [Type Documentation](#type-documentation)
- [Schema Documentation](#schema-documentation)
- [Class Documentation](#class-documentation)
- [Component Documentation (React)](#component-documentation-react)
- [Hook Documentation](#hook-documentation)
- [Required Tags](#required-tags-typedoc-compatible-only)
- [Documentation Examples](#documentation-examples)
- [Documentation Coverage](#documentation-coverage)
- [Best Practices](#best-practices)

## Function Documentation

```typescript
/**
 * Create a new user account with validation
 * Returns undefined if validation fails or user already exists
 *
 * @param params - User creation parameters
 * @param params.name - Full name of the user
 * @param params.email - Valid email address
 * @param params.role - User role (defaults to 'user')
 * @returns Created user object or undefined on failure
 */
export const createUser = async (params: CreateUserParams): Promise<User | undefined> => {
    const { name, email, role = 'user' } = params
    // Implementation
}
```

**Pattern**: Brief description, parameter details with destructured object parameters, explicit return documentation.

## Type Documentation

```typescript
/**
 * User account representation
 * Stored in PostgreSQL users table
 */
export type User = {
    /** Unique identifier (UUID v4) */
    id: string

    /** Full name of the user */
    name: string

    /** Email address (unique, validated) */
    email: string

    /** User role for access control */
    role: 'admin' | 'user' | 'guest'

    /** Account creation timestamp */
    createdAt: Date
}
```

**Pattern**: Type description, then brief field descriptions. Include constraints and validation details.

## Schema Documentation

```typescript
/**
 * User validation schema
 * Corresponds to User type and PostgreSQL schema
 */
export const userSchema = z.object({
    /**
     * Unique identifier
     * @example "550e8400-e29b-41d4-a716-446655440000"
     */
    id: z.string(),

    /**
     * Full name (2-100 characters)
     * @example "John Doe"
     */
    name: z.string().min(2).max(100),

    /**
     * Email address (validated format)
     * @example "john@example.com"
     */
    email: z.string().email(),

    /**
     * User role for access control
     * @example "user"
     */
    role: z.enum(['admin', 'user', 'guest']),
})

export type User = z.infer<typeof userSchema>
```

**Pattern**: Schema-level and field-level documentation. Use `@example` for clarity. Document relationship to type.

## Class Documentation

```typescript
/**
 * Repository for user data access
 * Handles all database operations for User entities
 */
export class UserRepository {
    /**
     * Initialize repository with database connection
     *
     * @param db - Database instance
     */
    constructor(private readonly db: Database) {}

    /**
     * Find user by unique identifier
     *
     * @param id - User UUID
     * @returns User object or undefined if not found
     */
    async findById(id: string): Promise<User | undefined> {
        // Implementation
    }

    /**
     * Save user to database
     * Creates new user or updates existing one
     *
     * @param user - User object to save
     */
    async save(user: User): Promise<void> {
        // Implementation
    }
}
```

**Pattern**: Class purpose, constructor documentation, method descriptions with parameters and return values.

## Component Documentation (React)

```typescript
/**
 * User profile card component
 * Displays user information with avatar, name, email, and role
 *
 * @param props - Component props
 * @param props.user - User object to display
 * @param props.isEditable - Whether to show edit button
 * @param props.onEdit - Callback when edit button is clicked
 */
export const UserCard: FC<UserCardProps> = ({ user, isEditable, onEdit }) => {
    // Implementation
}

/**
 * Props for UserCard component
 */
export type UserCardProps = {
    /** User object to display */
    user: User

    /** Whether to show edit button */
    isEditable?: boolean

    /** Callback when edit button is clicked */
    onEdit?: () => void
}
```

**Pattern**: Component purpose, visual output description, destructured prop documentation. Separate props type with field descriptions.

## Hook Documentation

```typescript
/**
 * Hook to fetch and manage user data
 * Handles loading state, error handling, and automatic refetching
 *
 * Side effects:
 * - Fetches user data on mount
 * - Refetches when userId changes
 * - Caches result in React Query cache
 *
 * @param userId - User identifier to fetch
 * @returns User data, loading state, and error state
 */
export const useUser = (userId: string): UseUserResult => {
    // Implementation
}

/**
 * Return value for useUser hook
 */
export type UseUserResult = {
    /** User data (undefined if loading or error) */
    user: User | undefined

    /** Whether user data is being fetched */
    isLoading: boolean

    /** Error object if fetch failed */
    error: Error | undefined

    /** Function to manually refetch user data */
    refetch: () => void
}
```

**Pattern**: Hook purpose, behavior description, side effects, parameter details, return value structure.

## Required Tags (TypeDoc-compatible only)

Use these TypeDoc-compatible tags:

- `@param` - Parameter description (with destructured object parameters)
- `@returns` - Return value description (be explicit about undefined/null)
- `@example` - Usage example (especially for schemas and complex types)
- `@throws` - Exception description (use sparingly, prefer returning undefined)
- `@deprecated` - Mark deprecated APIs (include migration path)

**Do NOT use these tags** (not supported by TypeDoc or maintained in git):

- `@author` - Maintained in git history
- `@version` - Maintained in package.json and git tags
- `@since` - Maintained in git history
- `@see` - Use markdown links instead: `See [UserService](./user-service.ts)`
- JSDoc-specific tags not supported by TypeDoc

## Documentation Examples

### Example Tag

```typescript
/**
 * Format date to ISO string
 *
 * @param date - Date to format
 * @returns ISO 8601 formatted string
 * @example
 * ```typescript
 * formatDate(new Date('2025-01-15'))
 * // Returns: "2025-01-15T00:00:00.000Z"
 * ```
 */
export const formatDate = (date: Date): string => {
    return date.toISOString()
}
```

**Pattern**: Use `@example` with code blocks for complex or non-obvious functions.

### Throws Tag

```typescript
/**
 * Divide two numbers
 *
 * @param a - Dividend
 * @param b - Divisor
 * @returns Result of division
 * @throws {Error} When divisor is zero
 */
export const divide = (a: number, b: number): number => {
    if (b === 0) throw new Error('Division by zero')
    return a / b
}
```

**Pattern**: Use `@throws` when function throws exceptions. Include exception type and condition.

### Deprecated Tag

```typescript
/**
 * Get user by ID (legacy implementation)
 *
 * @param id - User identifier
 * @returns User object
 * @deprecated Use `findUserById` from UserRepository instead
 */
export const getUser = async (id: string): Promise<User> => {
    // Implementation
}
```

**Pattern**: Use `@deprecated` with migration guidance for deprecated APIs.

## Documentation Coverage

Document all public APIs:

- ✅ Exported functions
- ✅ Exported types and interfaces
- ✅ Exported classes and public methods
- ✅ Public component props
- ✅ Hook parameters and return values
- ✅ Zod schemas and inferred types

Do NOT document:

- ❌ Private/internal functions
- ❌ Implementation details
- ❌ Obvious getters/setters
- ❌ Self-explanatory one-line utilities

## Best Practices

### Be Concise

```typescript
// ✅ Good: Concise and clear
/**
 * Calculate user age from birthdate
 *
 * @param birthDate - User's date of birth
 * @returns Age in years
 */
export const calculateAge = (birthDate: Date): number => {
    // Implementation
}

// ❌ Bad: Verbose and redundant
/**
 * This function takes a birthdate as input and calculates the age
 * of the user by subtracting the birth year from the current year.
 * It's useful when you need to determine how old someone is based
 * on when they were born.
 *
 * @param birthDate - This parameter represents the date when the user was born
 * @returns This function returns a number that represents the age in years
 */
export const calculateAge = (birthDate: Date): number => {
    // Implementation
}
```

**Pattern**: Be concise. Avoid redundancy. Types already provide parameter/return information.

### Document the "Why", Not the "What"

```typescript
// ✅ Good: Explains purpose and constraints
/**
 * Sanitize user input to prevent XSS attacks
 * Strips HTML tags and encodes special characters
 *
 * @param input - Raw user input
 * @returns Sanitized safe string
 */
export const sanitizeInput = (input: string): string => {
    // Implementation
}

// ❌ Bad: Just describes what code does
/**
 * Takes a string and returns a string
 *
 * @param input - A string
 * @returns A string
 */
export const sanitizeInput = (input: string): string => {
    // Implementation
}
```

**Pattern**: Explain purpose, constraints, and context. Don't just restate the signature.

### TypeDoc Comment Format

Always use `/** */` format (not `//` or `/* */`):

```typescript
// ✅ Good: JSDoc format
/**
 * Function description
 */
export const myFunction = (): void => {}

// ❌ Bad: Regular comments
// Function description
export const myFunction = (): void => {}

/* Function description */
export const myFunction = (): void => {}
```

**Pattern**: Always use `/** */` for documentation comments. TypeDoc only processes this format.
