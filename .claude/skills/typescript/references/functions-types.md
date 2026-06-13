# Functions & Types Patterns

Type-safe function and type definition patterns.

## Type Definition Patterns

### Types Over Interfaces

```typescript
// ✅ Preferred: Use types
type User = {
    id: string
    name: string
    email: string
}

type UserWithRole = User & {
    role: string
}

// ❌ Avoid: Interfaces (unless specifically needed for declaration merging)
interface User {
    id: string
    name: string
}
```

**Pattern**: Prefer `type` over `interface` for better composition and consistency.

### Readonly Properties

```typescript
type Config = {
    readonly apiUrl: string
    readonly timeout: number
}

const config: Config = {
    apiUrl: 'https://api.example.com',
    timeout: 5000,
}

// config.apiUrl = 'https://other.com' // ❌ Error: Cannot assign to 'apiUrl'
```

**Pattern**: Use `readonly` for immutable properties to enforce data integrity.

## Function Patterns

### Arrow Functions with Explicit Types

```typescript
export const getUser = async (userId: string): Promise<User | undefined> => {
    // Implementation
}

export const formatDate = (date: Date): string => {
    return date.toISOString().split('T')[0]
}

export const validateEmail = (email: string): boolean => {
    return /^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(email)
}
```

**Pattern**: All functions are arrow functions. Return types always explicit. Start function names with verbs.

### Short, Focused Functions

```typescript
// ✅ Good: Single purpose, <20 lines
export const createUserDto = (user: User): UserDto => ({
    id: user.id,
    name: user.name,
    email: user.email,
})

// ❌ Bad: Multiple purposes, too long
export const processUser = (user: User): void => {
    // Validate
    if (!user.email) throw new Error('Email required')
    if (!user.name) throw new Error('Name required')

    // Transform
    const dto = { id: user.id, name: user.name, email: user.email }

    // Save
    database.save(dto)

    // Send email
    emailService.send(dto.email, 'Welcome!')
}
```

**Pattern**: Functions should be <20 lines with single purpose. Extract utilities when needed.

### Early Returns

```typescript
// ✅ Good: Early returns, no nesting
export const calculateDiscount = (price: number, userType: string): number => {
    if (price <= 0) return 0
    if (userType === 'premium') return price * 0.2
    if (userType === 'regular') return price * 0.1
    return 0
}

// ❌ Bad: Nested conditions
export const calculateDiscount = (price: number, userType: string): number => {
    if (price > 0) {
        if (userType === 'premium') {
            return price * 0.2
        } else if (userType === 'regular') {
            return price * 0.1
        } else {
            return 0
        }
    } else {
        return 0
    }
}
```

**Pattern**: Use early returns to reduce nesting and improve readability.

### Higher-Order Functions

```typescript
// ✅ Good: Use map, filter, reduce
export const getActiveUserEmails = (users: User[]): string[] => {
    return users
        .filter(user => user.isActive)
        .map(user => user.email)
}

// ❌ Bad: Manual iteration
export const getActiveUserEmails = (users: User[]): string[] => {
    const emails: string[] = []
    for (let i = 0; i < users.length; i++) {
        if (users[i].isActive) {
            emails.push(users[i].email)
        }
    }
    return emails
}
```

**Pattern**: Prefer higher-order functions (map, filter, reduce) over manual iteration. Use arrow functions for simple transformations.

### RO-RO Pattern (Receive Object, Return Object)

```typescript
type CreateUserParams = {
    name: string
    email: string
    role?: string
}

type CreateUserResult = {
    user: User
    token: string
}

export const createUser = async (params: CreateUserParams): Promise<CreateUserResult> => {
    const { name, email, role = 'user' } = params

    const user = await database.createUser({ name, email, role })
    const token = generateToken(user.id)

    return { user, token }
}
```

**Pattern**: For functions with multiple parameters (more than 3), use object parameters. For complex results, return objects. Define dedicated types for both.

### Default Parameters

```typescript
// ✅ Good: Default parameter values
export const fetchUsers = async (limit: number = 10, offset: number = 0): Promise<User[]> => {
    return database.query({ limit, offset })
}

// ❌ Bad: Manual null/undefined checks
export const fetchUsers = async (limit?: number, offset?: number): Promise<User[]> => {
    const actualLimit = limit ?? 10
    const actualOffset = offset ?? 0
    return database.query({ limit: actualLimit, offset: actualOffset })
}
```

**Pattern**: Use default parameter values instead of checking for null/undefined.

## Import/Export Patterns

### Named Exports Only

```typescript
// ✅ Good: Named exports
export const UserService = { ... }
export type { User, UserDto }

// ❌ Bad: Default exports
export default UserService
```

**Pattern**: Always use named exports. No default exports. Improves discoverability and refactoring.

### Barrel Exports (index.ts)

```typescript
// src/domain/user/index.ts
export { createUser, updateUser, deleteUser } from './user-service'
export type { User, UserDto, CreateUserParams } from './types'
```

**Pattern**: Each module exports through `index.ts`. Type exports separate from value exports. Do not use for NestJS module.

### Single Level of Abstraction

```typescript
// ✅ Good: Single level of abstraction
export const processOrder = async (orderId: string): Promise<void> => {
    const order = await fetchOrder(orderId)
    const payment = await processPayment(order)
    await sendConfirmation(order, payment)
}

const fetchOrder = async (orderId: string): Promise<Order> => {
    // Low-level database details
}

const processPayment = async (order: Order): Promise<Payment> => {
    // Low-level payment gateway details
}

const sendConfirmation = async (order: Order, payment: Payment): Promise<void> => {
    // Low-level email details
}

// ❌ Bad: Mixed abstraction levels
export const processOrder = async (orderId: string): Promise<void> => {
    const order = await database.query('SELECT * FROM orders WHERE id = ?', [orderId])
    await paymentGateway.charge({ amount: order.total, card: order.cardToken })
    await emailService.send({ to: order.email, subject: 'Order Confirmed' })
}
```

**Pattern**: Each function should operate at a single level of abstraction. Extract lower-level details into separate functions.
