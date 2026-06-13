# Error Handling Patterns

Exception handling, error context, and graceful degradation patterns.

## When to Throw vs Return Undefined

### Return Undefined for Expected Failures

```typescript
// ✅ Good: Return undefined for expected failures
export const findUser = async (id: string): Promise<User | undefined> => {
    try {
        const response = await fetch(`/api/users/${id}`)
        if (!response.ok) return undefined

        const data = await response.json()
        const result = userSchema.safeParse(data)

        return result.success ? result.data : undefined
    } catch (error) {
        console.error('Error fetching user:', error)
        return undefined
    }
}

// Usage: Graceful handling
const user = await findUser(id)
if (!user) {
    console.log('User not found')
    return
}
```

**Pattern**: API functions return `undefined` for expected failures (not found, validation failed, network error). Errors logged with context.

### Throw Errors for Unexpected Failures

```typescript
// ✅ Good: Throw for unexpected/programmer errors
export const useAuth = (): AuthContext => {
    const context = useContext(AuthContext)
    if (!context) {
        throw new Error('useAuth must be used within AuthProvider')
    }
    return context
}

export const divide = (a: number, b: number): number => {
    if (b === 0) {
        throw new Error('Division by zero')
    }
    return a / b
}
```

**Pattern**: Throw errors for unexpected failures (missing context, invariant violations, programmer errors).

## Exception Handling Patterns

### Catch to Fix Expected Problems

```typescript
// ✅ Good: Catch to fix expected problems
export const loadConfig = (): Config => {
    try {
        const content = fs.readFileSync('config.json', 'utf-8')
        return JSON.parse(content)
    } catch (error) {
        // Expected: Config file might not exist, use defaults
        console.warn('Config file not found, using defaults')
        return DEFAULT_CONFIG
    }
}
```

**Pattern**: Catch exceptions to fix expected problems (missing files, parse errors) and provide defaults.

### Catch to Add Context

```typescript
// ✅ Good: Catch to add context, then re-throw
export const processFile = async (filePath: string): Promise<void> => {
    try {
        const content = await readFile(filePath)
        await validateContent(content)
        await saveToDatabase(content)
    } catch (error) {
        throw new Error(`Failed to process file ${filePath}: ${error}`)
    }
}
```

**Pattern**: Catch exceptions to add context (file path, user ID, etc.), then re-throw.

## Error Context Patterns

### Provide Detailed Error Messages

```typescript
// ✅ Good: Detailed error messages
export const createUser = async (email: string): Promise<User> => {
    if (!email) {
        throw new Error('Email is required for user creation')
    }

    const existing = await findUserByEmail(email)
    if (existing) {
        throw new Error(`User with email ${email} already exists`)
    }

    // ... create user
}

// ❌ Bad: Generic error messages
export const createUser = async (email: string): Promise<User> => {
    if (!email) throw new Error('Invalid input')
    const existing = await findUserByEmail(email)
    if (existing) throw new Error('User exists')
}
```

**Pattern**: Error messages should include context about what failed and why. Include variable values when relevant.

### Custom Error Classes

```typescript
// ✅ Good: Custom error classes for domain errors
export class UserNotFoundError extends Error {
    constructor(public readonly userId: string) {
        super(`User not found: ${userId}`)
        this.name = 'UserNotFoundError'
    }
}

export class ValidationError extends Error {
    constructor(
        public readonly field: string,
        public readonly value: unknown,
    ) {
        super(`Validation failed for ${field}: ${value}`)
        this.name = 'ValidationError'
    }
}

// Usage with type guards
try {
    await updateUser(userId, data)
} catch (error) {
    if (error instanceof UserNotFoundError) {
        return res.status(404).json({ error: 'User not found' })
    }
    if (error instanceof ValidationError) {
        return res.status(400).json({ error: error.message, field: error.field })
    }
    throw error // Re-throw unexpected errors
}
```

**Pattern**: Create custom error classes for domain-specific errors. Include relevant context as properties.

## Graceful Degradation

### Handle Missing Data Gracefully

```typescript
// ✅ Good: Graceful degradation
export const getUserProfile = async (userId: string): Promise<UserProfile> => {
    const user = await findUser(userId)
    const avatar = await fetchAvatar(userId) // May fail
    const preferences = await loadPreferences(userId) // May fail

    return {
        user: user ?? DEFAULT_USER,
        avatar: avatar ?? DEFAULT_AVATAR,
        preferences: preferences ?? DEFAULT_PREFERENCES,
    }
}

// ❌ Bad: Fail completely if anything fails
export const getUserProfile = async (userId: string): Promise<UserProfile> => {
    const user = await findUser(userId)
    const avatar = await fetchAvatar(userId)
    const preferences = await loadPreferences(userId)

    return { user, avatar, preferences } // Throws if any fails
}
```

**Pattern**: For non-critical data, provide defaults instead of failing. Only fail for critical operations.

## Type Guard Filtering

### Filter Undefineds from Arrays

```typescript
// ✅ Good: Type guard filtering
const results: (User | undefined)[] = await Promise.all(
    userIds.map(id => findUser(id))
)

const users: User[] = results.filter((user): user is User => user !== undefined)
```

**Pattern**: Use type predicates to filter undefineds and refine array types after Promise.all().

### Filter Failed Results

```typescript
type Success<T> = { success: true; data: T }
type Failure = { success: false; error: string }
type Result<T> = Success<T> | Failure

const results: Result<User>[] = await Promise.all(
    userIds.map(async id => {
        try {
            const user = await findUser(id)
            return user ? { success: true, data: user } : { success: false, error: 'Not found' }
        } catch (error) {
            return { success: false, error: String(error) }
        }
    })
)

const users: User[] = results
    .filter((result): result is Success<User> => result.success)
    .map(result => result.data)

const errors: string[] = results
    .filter((result): result is Failure => !result.success)
    .map(result => result.error)
```

**Pattern**: Use discriminated unions with type guards to separate successes from failures.
