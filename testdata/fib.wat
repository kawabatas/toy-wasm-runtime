(module
  (func $fib (param $n i32) (result i32)
    local.get $n
    i32.const 2
    i32.lt_s
    (if (result i32)
      (then
        (local.get $n)
      )
      (else
        ;; n-1
        local.get $n
        i32.const 1
        i32.sub
        call $fib
        ;; n-2
        local.get $n
        i32.const 2
        i32.sub
        call $fib
        ;; (n-1)+(n-2)
        i32.add
      )
    )
  )
  (export "fib" (func $fib))
)
