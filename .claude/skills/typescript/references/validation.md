# Validation & Schema Patterns

Data validation, type guards, and immutability patterns.

## Zod Schema Patterns

### Schema-First Type Definition

```typescript
import { z } from 'zod'

export const userSchema = z.object({
    id: z.string(),
    name: z.string(),
    email: z.email(),
    role: z.enum(['admin', 'user', 'guest']),
    createdAt: z.coerce.date(),
})

export type User = z.infer<typeof userSchema>
```

**Pattern**: Define Zod schema first, export both schema and inferred type.

### Safe Parsing with Error Handling

```typescript
// ✅ Good: Safe parsing with error handling
export const parseUser = (data: unknown): User | undefined => {
    const result = userSchema.safeParse(data)

    if (!result.success) {
        console.error('Invalid user data:', result.error)
        return undefined
    }

    return result.data
}

// ❌ Bad: Direct parsing (throws errors)
export const parseUser = (data: unknown): User => {
    return userSchema.parse(data) // Throws on invalid data
}
```

**Pattern**: Always use `.safeParse()` with error handling. Never use `.parse()` directly.

### Optional and Nullable Fields

```typescript
export const userProfileSchema = z.object({
    bio: z.string().optional(),           // Field may be missing
})
```

**Pattern**: Use `.optional()` for fields that may not exist or empty.

## Type Guards

### Custom Type Guards

```typescript
type Success<T> = { success: true; data: T }
type Failure = { success: false; error: string }
type Result<T> = Success<T> | Failure

const isSuccess = <T>(result: Result<T>): result is Success<T> => {
    return result.success === true
}

// Usage
const result: Result<User> = await fetchUser(id)

if (isSuccess(result)) {
    console.log(result.data) // TypeScript knows this is Success<User>
} else {
    console.error(result.error) // TypeScript knows this is Failure
}
```

**Pattern**: Use type predicates (`result is Success<T>`) for custom type guards.

### Array Filtering with Type Guards

```typescript
const users: (User | undefined)[] = await Promise.all(
    ids.map(id => fetchUser(id))
)

const validUsers: User[] = users.filter((user): user is User => user !== undefined)
```

**Pattern**: Filter with type predicates to exclude undefineds and refine array types.

## Immutability Patterns

### Readonly Properties

```typescript
type User = {
    readonly id: string
    readonly email: string
    name: string // Mutable
}

const user: User = { id: '1', email: 'user@example.com', name: 'John' }

user.name = 'Jane' // ✅ OK
user.id = '2' // ❌ Error: Cannot assign to 'id'
user.email = 'other@example.com' // ❌ Error: Cannot assign to 'email'
```

**Pattern**: Use `readonly` for properties that should not change after initialization.

### As Const for Literals

```typescript
// ✅ Good: Use 'as const' for immutable literals
const ROLES = ['admin', 'user', 'guest'] as const
type Role = typeof ROLES[number] // 'admin' | 'user' | 'guest'

const CONFIG = {
    apiUrl: 'https://api.example.com',
    timeout: 5000,
} as const

// ❌ Bad: Mutable literals
const ROLES = ['admin', 'user', 'guest'] // string[]
const CONFIG = {
    apiUrl: 'https://api.example.com',
    timeout: 5000,
} // { apiUrl: string; timeout: number }
```

**Pattern**: Use `as const` for literal values that should be immutable and infer narrow types.

### Prefer Immutability

```typescript
// ✅ Good: Immutable data transformations
const addUser = (users: readonly User[], newUser: User): readonly User[] => {
    return [...users, newUser]
}

const updateUser = (users: readonly User[], id: string, updates: Partial<User>): readonly User[] => {
    return users.map(user => user.id === id ? { ...user, ...updates } : user)
}

// ❌ Bad: Mutable operations
const addUser = (users: User[], newUser: User): void => {
    users.push(newUser)
}

const updateUser = (users: User[], id: string, updates: Partial<User>): void => {
    const user = users.find(u => u.id === id)
    if (user) Object.assign(user, updates)
}
```

**Pattern**: Prefer immutable data transformations over mutations. Return new arrays/objects instead of modifying in place.

## Composite Types

### Avoid Primitive Obsession

```typescript
// ✅ Good: Encapsulate data in composite types
type Money = {
    readonly amount: number
    readonly currency: string
}

type Product = {
    readonly id: string
    readonly name: string
    readonly price: Money
}

const product: Product = {
    id: '1',
    name: 'Widget',
    price: { amount: 99.99, currency: 'USD' },
}

// ❌ Bad: Primitive obsession
type Product = {
    id: string
    name: string
    priceAmount: number
    priceCurrency: string
}
```

**Pattern**: Don't abuse primitive types. Encapsulate related data in composite types for better type safety and clarity.
