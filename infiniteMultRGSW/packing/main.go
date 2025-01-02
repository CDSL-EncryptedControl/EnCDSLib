package main

import (
	"fmt"
	"math"

	"time"

	utils "github.com/CDSL-EncryptedControl/2024SICE/utils"
	RGSW "github.com/CDSL-EncryptedControl/2024SICE/utils/core/RGSW"
	RLWE "github.com/CDSL-EncryptedControl/2024SICE/utils/core/RLWE"
	"github.com/tuneinsight/lattigo/v6/core/rgsw"
	"github.com/tuneinsight/lattigo/v6/core/rlwe"
	"github.com/tuneinsight/lattigo/v6/ring"
)

func main() {
	// *****************************************************************
	// ************************* User's choice *************************
	// *****************************************************************
	// ============== Encryption parameters ==============
	// Refer to ``Homomorphic encryption standard''
	params, _ := rlwe.NewParametersFromLiteral(rlwe.ParametersLiteral{
		// log2 of polynomial degree
		LogN: 11,
		// Size of ciphertext modulus (Q)
		LogQ: []int{56},
		// Size of plaintext modulus (P)
		LogP:    []int{42},
		NTTFlag: true,
	})
	fmt.Println("Degree of polynomials:", params.N())
	fmt.Println("Ciphertext modulus:", params.QBigInt())
	fmt.Println("Ciphertext modulus:", params.PBigInt())

	// ============== Plant model ==============
	A := [][]float64{
		{0.9984, 0, 0.0042, 0},
		{0, 0.9989, 0, -0.0033},
		{0, 0, 0.9958, 0},
		{0, 0, 0, 0.9967},
	}
	B := [][]float64{
		{0.0083, 0},
		{0, 0.0063},
		{0, 0.0048},
		{0.0031, 0},
	}
	C := [][]float64{
		{0.5, 0, 0, 0},
		{0, 0.5, 0, 0},
	}
	// Plant initial state
	xp0 := []float64{
		1,
		1,
		1,
		1,
	}

	// ============== Pre-designed controller ==============
	// F must be an integer matrix
	F := [][]float64{
		{-1, 0, 0, 0},
		{0, 0, 0, 0},
		{0, 0, 2, 0},
		{0, 0, 0, 1},
	}
	G := [][]float64{
		{0.7160, -0.3828},
		{-0.8131, -1.4790},
		{0.6646, 1.1860},
		{0.0181, -0.0060},
	}
	R := [][]float64{
		{-1.7396, 0.3476},
		{0.2588, 1.3226},
		{0.5115, 2.4668},
		{0.0122, 0.0030},
	}
	H := [][]float64{
		{-0.8829, 0.0445, -0.0533, -0.0855},
		{0.1791, 0.2180, -0.2738, 0.0180},
	}
	// Controller initial state
	xc0 := []float64{
		0.5,
		0.02,
		-1,
		0.9,
	}
	// dimensions
	n := len(F)
	m := len(H)
	p := len(G[0])

	// ============== Quantization parameters ==============
	s := 1 / 10000.0
	L := 1 / 1000.0
	r := 1 / 1000.0
	fmt.Printf("Scaling parameters 1/L: %v, 1/s: %v, 1/r: %v \n", 1/L, 1/s, 1/r)
	// *****************************************************************
	// *****************************************************************

	// ============== Encryption settings ==============
	// Set parameters
	levelQ := params.QCount() - 1
	levelP := params.PCount() - 1
	ringQ := params.RingQ()

	// Compute tau
	// least power of two greater than n, p_, and m
	maxDim := math.Max(math.Max(float64(n), float64(m)), float64(p))
	tau := int(math.Pow(2, math.Ceil(math.Log2(maxDim))))

	// Generate DFS index
	dfsId := make([]int, tau)
	for i := 0; i < tau; i++ {
		dfsId[i] = i
	}

	tmp := make([]int, tau)
	for i := 1; i < tau; i *= 2 {
		id := 0
		currBlock := tau / i
		nextBlock := currBlock / 2
		for j := 0; j < i; j++ {
			for k := 0; k < nextBlock; k++ {
				tmp[id] = dfsId[j*currBlock+2*k]
				tmp[nextBlock+id] = dfsId[j*currBlock+2*k+1]
				id++
			}
			id += nextBlock
		}

		for j := 0; j < tau; j++ {
			dfsId[j] = tmp[j]
		}
	}

	// Generate monomials for unpack
	logn := int(math.Log2(float64(tau)))
	monomials := make([]ring.Poly, logn)
	for i := 0; i < logn; i++ {
		monomials[i] = ringQ.NewPoly()
		idx := params.N() - params.N()/(1<<(i+1))
		monomials[i].Coeffs[0][idx] = 1
		ringQ.MForm(monomials[i], monomials[i])
		ringQ.NTT(monomials[i], monomials[i])
	}

	// Generate Galois elements
	galEls := make([]uint64, int(math.Log2(float64(tau))))
	for i := 0; i < int(math.Log2(float64(tau))); i++ {
		galEls[i] = uint64(tau/int(math.Pow(2, float64(i))) + 1)
	}

	// Generate keys
	kgen := rlwe.NewKeyGenerator(params)
	sk := kgen.GenSecretKeyNew()
	rlk := kgen.GenRelinearizationKeyNew(sk)
	evkRGSW := rlwe.NewMemEvaluationKeySet(rlk)
	evkRLWE := rlwe.NewMemEvaluationKeySet(rlk, kgen.GenGaloisKeysNew(galEls, sk)...)

	// Define encryptor and evaluator
	encryptorRLWE := rlwe.NewEncryptor(params, sk)
	decryptorRLWE := rlwe.NewDecryptor(params, sk)
	encryptorRGSW := rgsw.NewEncryptor(params, sk)
	evaluatorRGSW := rgsw.NewEvaluator(params, evkRGSW)
	evaluatorRLWE := rlwe.NewEvaluator(params, evkRLWE)

	// ==============  Encryption of controller ==============
	// Quantization
	Gbar := utils.ScalarMatMult(1/s, G)
	Rbar := utils.ScalarMatMult(1/s, R)
	Hbar := utils.ScalarMatMult(1/s, H)

	// Encryption
	// Dimension: 1-by-(# of columns)
	ctF := RGSW.EncPack(F, tau, encryptorRGSW, levelQ, levelP, ringQ, params)
	ctG := RGSW.EncPack(Gbar, tau, encryptorRGSW, levelQ, levelP, ringQ, params)
	ctH := RGSW.EncPack(Hbar, tau, encryptorRGSW, levelQ, levelP, ringQ, params)
	ctR := RGSW.EncPack(Rbar, tau, encryptorRGSW, levelQ, levelP, ringQ, params)

	// ============== Simulation ==============
	// Number of simulation steps
	iter := 1000
	fmt.Printf("Number of iterations: %v", iter)

	// *****************
	// 1) Plant + unencrypted (original) controller
	// *****************

	// State and output storage
	yUnenc := [][]float64{}
	uUnenc := [][]float64{}
	xcUnenc := [][]float64{}
	xpUnenc := [][]float64{}

	xpUnenc = append(xpUnenc, xp0)
	xcUnenc = append(xcUnenc, xc0)

	// Plant state
	xp := xp0
	// Controller state
	xc := xc0

	for i := 0; i < iter; i++ {
		y := utils.MatVecMult(C, xp)
		u := utils.MatVecMult(H, xc)
		xp = utils.VecAdd(utils.MatVecMult(A, xp), utils.MatVecMult(B, u))
		xc = utils.VecAdd(utils.MatVecMult(F, xc), utils.MatVecMult(G, y))
		xc = utils.VecAdd(xc, utils.MatVecMult(R, u))

		yUnenc = append(yUnenc, y)
		uUnenc = append(uUnenc, u)
		xcUnenc = append(xcUnenc, xc)
		xpUnenc = append(xpUnenc, xp)
	}

	// *****************
	// 2) Plant + encrypted controller
	// *****************

	// State and output storage
	yEnc := [][]float64{}
	uEnc := [][]float64{}
	xpEnc := [][]float64{}
	xpEnc = append(xpEnc, xp0)

	// Plant state
	xp = xp0

	// Dimension: 1-by-(# of elements)
	xcScale := utils.ScalarVecMult(1/(r*s), xc0)
	xcCt := RLWE.Enc(xcScale, 1/L, *encryptorRLWE, ringQ, params)
	xcCtPack := rlwe.NewCiphertext(params, xcCt[0].Degree(), xcCt[0].Level())

	// For time check
	period := make([][]float64, iter)
	startPeriod := make([]time.Time, iter)

	for i := 0; i < iter; i++ {
		// **** Sensor ****
		// Plant output
		y := utils.MatVecMult(C, xp)

		startPeriod[i] = time.Now()

		// Quantize - encrypt
		yRound := utils.RoundVec(utils.ScalarVecMult(1/r, y))
		yCt := RLWE.Enc(yRound, 1/L, *encryptorRLWE, ringQ, params)

		// **** Encrypted Controller ****
		// Comput output
		uCt := RGSW.MultPack(xcCt, ctH, evaluatorRGSW, ringQ, params)

		// **** Actuator ****
		// Unpack - decrypt
		uCtUnpack := RLWE.UnpackCt(uCt, m, tau, evaluatorRLWE, ringQ, monomials, params)
		u := RLWE.Dec(uCtUnpack, *decryptorRLWE, r*s*s*L, ringQ, params)

		// Re-encrypt output
		uReEnc := RLWE.Enc(u, 1/(r*L), *encryptorRLWE, ringQ, params)

		// **** Encrypted Controller ****
		// State update
		FxCt := RGSW.MultPack(xcCt, ctF, evaluatorRGSW, ringQ, params)
		GyCt := RGSW.MultPack(yCt, ctG, evaluatorRGSW, ringQ, params)
		RuCt := RGSW.MultPack(uReEnc, ctR, evaluatorRGSW, ringQ, params)
		xcCtPack = RLWE.Add(FxCt, GyCt, params)
		xcCtPack = RLWE.Add(xcCtPack, RuCt, params)
		xcCt = RLWE.UnpackCt(xcCtPack, n, tau, evaluatorRLWE, ringQ, monomials, params)

		period[i] = []float64{float64(time.Since(startPeriod[i]).Microseconds()) / 1000}

		// **** Plant ****
		// State update
		xp = utils.VecAdd(utils.MatVecMult(A, xp), utils.MatVecMult(B, u))

		// Save data
		yEnc = append(yEnc, y)
		uEnc = append(uEnc, u)
		xpEnc = append(xpEnc, xp)
	}

	avgPeriod := utils.Average(utils.MatToVec(period))
	fmt.Println("Average elapsed time for a control period:", avgPeriod, "ms")

	// Compare plant input between 1) and 2)
	uDiff := make([][]float64, iter)
	for i := range uDiff {
		uDiff[i] = []float64{utils.Vec2Norm(utils.VecSub(uUnenc[i], uEnc[i]))}
	}

	// Export data ===============================================================

	// =========== Export data ===========

	// Plant state equipped with encrypted controller
	utils.DataExport(xpEnc, "./state.csv")

	// Plant intput from encrypted controller
	utils.DataExport(uEnc, "./uEnc.csv")

	// Plant output with encrypted controller
	utils.DataExport(yEnc, "./yEnc.csv")

	// Performance of encrypted controller
	utils.DataExport(uDiff, "./uDiff.csv")

	// Elapsed time
	utils.DataExport(period, "./period.csv")
}