# CDSL: Cryptography for Dynamic Systems Library

Still developing readme section

CDSL provides codes for implementing secure dynamic systems based on modern cryptography.
The library features linear dynamic controllers operating over homomorphically encrypted data implemented using [Lattigo](https://github.com/tuneinsight/lattigo) version 6.1.0.
The encrypted controllers are designed based on the state-of-the-art methods developed by CDSL, [SNU](https://post.cdsl.kr/) and [SEOULTECH](https://junsookim4.wordpress.com/).


---

### Overview


Given a plant 

$$
\begin{aligned}
x_p(t+1) &= Ax_p(t) + Bu(t), \quad x_p(0) = x_p^{\mathsf{ini}} \\
y(t) &= Cx_p(t)
\end{aligned}
$$

and a pre-designed stabilizng controller (which is controllable and observable)

$$
\begin{aligned}
x(t+1) &= Fx(t) + Gy(t), \quad x(0) = x^{\mathsf{ini}} \\
u(t) &= Hx(t)
\end{aligned}
$$

this code provides two methods to operate the pre-designed dynamic controller over encrypted data, using a Ring-LWE based cryptosystem. 


- `ctrRGSW` [1]: Supports unlimited number of recursive homomorhpic multiplications without the use of bootstrapping. More specifically, the encrypted controller state is recursively multiplied to the encrypted state matrix without decryption. The effect of error growth is suppressed by the stability of the closed-loop system. 
    - `ctrRGSW/noPacking`: Naive implementation that does not use packing. 
    - `ctrRGSW/packing`: A novel "coefficient packing" technique is applied, resulting in enhanced computation speed and memory efficiency   
    - `ctrRGSW/conversion.m`: Converts the state matrix of the controller into integers based on the apporach of [2]:
       - Given $F$ and $H$, it finds an appropriate $R$ such that $F-RH$ is an integer matrix. Then, the state dynamics of the controller can be rewritten as

$$
x(t+1) = (F-RH)x(t) + Gy(t) + Ru(t)
$$

       - regarding $u(t)$ as a fed-back input.

- `ctrRLWE` [2]: 
 

---

### Files
There are two files. 
1. `Ring-GSW.go` (without packing. Section 3 of [1])
2. `Ring-GSW_Packed.go` (with packing. Section 4 of [1])

When running one file, please comment out the other one. 
Then run

```
go run Ring-GSW.go  
```
or
```
go run Ring-GSW_Packed.go  
```
on the terminal.

---

### Set parameters 

* `rlwe.NewParametersFromLiteral`: Ring-LWE parameters (LogN = 11 and LogQ = 54 gives $N=2^{11}$ and some prime $q$ such that $q \approx 2^{54}$)

* `s`, `L`, and `r`: Scale factors 

* `iter`: Number of iterations for simulation 

* `A`, `B`, and `C`: State space matrices of the discrete time plant written by

> $x(t+1) = Ax(t) + Bu(t), \quad y(t) = Cx(t)$

* `F`, `G`, `R` and `H`: State space matrices of the discrete time controller. 
Given a controller of the form 
> $x(t+1) = Kx(t) + Gy(t), \quad u(t) = Hx(t)$

one can regard $u(t)$ as a fed-back input and design $R$, so that the state matrix $F:=K-RH$ of
> $x(t+1) = (K-RH)x(t) + Gy(t)+Ru(t), \quad u(t) = Hx(t)$

consists of integers. More details can be found in Section 5 of [1] or Lemma 1 of [2].

* `xPlantInit`, `xContInit`: Initial conditions of the plant and the controller

* `tau`: Least power of two greater than the dimensions of the state, output, and input of the controller (Only used in `Ring-GSW_Packed.go`)

---

### References
[1] [Y. Jang, J. Lee, S. Min, H. Kwak, J. Kim, and Y. Song, "Ring-LWE based encrypted controller with unlimited number of recursive multiplications and effect of error growth," 2024, arXiv:2406.14372.](https://arxiv.org/abs/2406.14372)

[2] [J. Kim, H. Shim, and K. Han, "Dynamic controller that operates over homomorphically encrypted data for infinite time horizon," _IEEE Trans. Autom. Control_, vol. 68, no. 2, pp. 660-672, 2023.](https://ieeexplore.ieee.org/abstract/document/9678042)

[3] [J. Lee, D. Lee, J. Kim, and H. Shim, "Encrypted dynamic control exploiting limited number of multiplications and a method using RLWE-based cryptosystem," _IEEE Trans. Syst. Man. Cybern.: Syst._, vol. 55, no. 1, pp. 158-169, 2025.](https://ieeexplore.ieee.org/abstract/document/10730788)

