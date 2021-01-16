# Coupon Codes 

## Abstract

Enable users to apply a promotional code to their account so that they can automatically receive the free credit associated with that code.

Summary: When Sales/Marketing whats to run a promotion or hand out a specific amount of free credit to a group of people, we want to be able to generate a coupon code that a user could enter into their account, so that specific free credits and expiration of the credits would be automatically applied to that user’s account.

## Background


## Design

There are 2 high level aspects of this design:

### Generating Coupon Codes

The satellite admin can generate coupon codes from the admin UI. Once generated, these codes can be applied to any user's account.

### Applying a Coupon Code to a User

On account creation or from the payments panel, a user should be able to insert a coupon code and immediately have a corresponding coupon applied to their account. If the user still has the default free-credits coupon, this coupon is removed and replaced with the new one.

## Rationale

In the design detailed below, one additional database table is defined. Each row of this table defines a coupon template and associates it with a unique coupon code (name). Since a user's coupons are already integrated into our billing logic, no additional code should be necessary for generating user invoices. 

## Implementation

### Database Layer

We will need a new table for coupon codes and their coupon templates, called `coupon_codes`. The table will have the following fields:
```
model coupon_code (
    key    id
    unique name

    field id          blob
    field name        text
    field amount      int64
    field duration    int64
    field description text
    field type        int

    field created_at timestamp ( autoinsert )
)
```
`name` must be a unique field for this table.

A nullable field will need to be added to the `coupons` table, allowing a coupon to be optionally linked to a coupon code:

```
model coupon (
    ...
    field coupon_code_name text  ( nullable )
    ...
)
```

### Service

The service layer will communicate with the database layer to add and remove coupon codes as an admin, or apply a coupon code as a user.

Here is a type definitions for the service layer:
```
type CouponCode struct {
    Name        string
    Duration    int // months
    Amount      int
    Description string
    Type        coupons.CouponType
}
```

It will require the following interface:

```
AddCouponCode(newCouponCode CouponCode) error
DeleteCouponCode(couponCodeName string) error

ApplyCouponCodeToUser(userID uuid.UUID, couponCodeName string, now time.Time) error
```

* `AddCouponCode` - insert a new row into the `coupon_codes` table with the provided values. Return an error if a coupon code with the provided `name` already exists.
* `DeleteCouponCode` - first, set any coupons in the `coupons` table with a matching `coupon_code_name` to be invalid or expired (this will require adding functionality to the coupons service/db layer). Then, delete the coupon code from the `coupon_codes` table.
* `ApplyCouponCodeToUser` - Look up the row corresponding to the provided coupon code name, then create a new coupon matching that template for the specified user. If the user has a default coupon applied, set this coupon to invalid or expired (maybe add a new state to represent this situation). 

### UI
#### **Admin**

The satellite admin should be able to add new coupon codes from the UI. There should be fields for name, amount, duration, and description.

It should be possible to delete existing coupon codes from the UI.

Editing existing coupon codes is not necessary for a minimal implementation, but in the future, we may want the ability to edit coupon codes from the UI.

The admin UI currently has no graphical interface. A minimal implementation can use the existing admin interface, but eventually we want non-developers with permission to be able to easily add and remove coupon codes.

#### **User**
There are a couple options we have from a UI perspective here. Either a user can create a coupon code on account creation, or add it from the payments panel in the webapp. We may even end up having both enabled or AB test one vs. the other. Either way, all that needs to be done is make a call to `service.ApplyCouponCodeToUser` when a user attempts to apply a coupon code.

## Wrapup

The Satellite Web Team will take ownership of this project and is responsible for archiving this blueprint upon completion.

## Open issues

* Can a user delete a coupon code from their account?
* Is there existing documentation that needs to be updated? (add to wrapup)
* Is there new documentation that needs to be written? (add to wrapup)
