package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strconv"
	"sync"

	"github.com/gin-gonic/gin"
)

// Student struct
type Student struct {
	ID    int    `json:"id"`
	Name  string `json:"name"`
	Age   int    `json:"age"`
	Email string `json:"email"`
}

// Global student list and mutex for concurrency
var (
	students []Student
	mu       sync.Mutex
	nextID   = 1
)

func main() {
	router := gin.Default()

	// Define API endpoints
	router.POST("/students", createStudent)
	router.GET("/students", getAllStudents)
	router.GET("/students/:id", getStudentByID)
	router.PUT("/students/:id", updateStudent)
	router.DELETE("/students/:id", deleteStudent)
	router.GET("/students/:id/summary", getStudentSummary) // New endpoint for summary

	log.Fatal(router.Run(":8080"))
}

// createStudent handles POST /students
func createStudent(c *gin.Context) {
	var newStudent Student
	if err := c.ShouldBindJSON(&newStudent); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Input validation
	if newStudent.Name == "" || newStudent.Age <= 0 || newStudent.Email == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input data"})
		return
	}

	mu.Lock()
	newStudent.ID = nextID
	nextID++
	students = append(students, newStudent)
	mu.Unlock()

	c.JSON(http.StatusCreated, gin.H{
		"message": "Student created successfully",
		"student": newStudent,
	})
}

// getAllStudents handles GET /students
func getAllStudents(c *gin.Context) {
	mu.Lock()
	defer mu.Unlock()
	c.JSON(http.StatusOK, students)
}

// getStudentByID handles GET /students/:id
func getStudentByID(c *gin.Context) {
	idParam := c.Param("id")
	id, err := strconv.Atoi(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID"})
		return
	}

	mu.Lock()
	defer mu.Unlock()
	for _, student := range students {
		if student.ID == id {
			c.JSON(http.StatusOK, student)
			return
		}
	}

	c.JSON(http.StatusNotFound, gin.H{"error": "Student not found"})
}

// updateStudent handles PUT /students/:id
func updateStudent(c *gin.Context) {
	idParam := c.Param("id")
	id, err := strconv.Atoi(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID"})
		return
	}

	var updatedStudent Student
	if err := c.ShouldBindJSON(&updatedStudent); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Input validation
	if updatedStudent.Name == "" || updatedStudent.Age <= 0 || updatedStudent.Email == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input data"})
		return
	}

	mu.Lock()
	defer mu.Unlock()
	for i, student := range students {
		if student.ID == id {
			students[i] = updatedStudent
			students[i].ID = id
			c.JSON(http.StatusOK, gin.H{"message": "Student updated successfully"})
			return
		}
	}

	c.JSON(http.StatusNotFound, gin.H{"error": "Student not found"})
}

// deleteStudent handles DELETE /students/:id
func deleteStudent(c *gin.Context) {
	idParam := c.Param("id")
	id, err := strconv.Atoi(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID"})
		return
	}

	mu.Lock()
	defer mu.Unlock()
	for i, student := range students {
		if student.ID == id {
			students = append(students[:i], students[i+1:]...)
			c.JSON(http.StatusOK, gin.H{"message": "Student deleted successfully"})
			return
		}
	}

	c.JSON(http.StatusNotFound, gin.H{"error": "Student not found"})
}

// getStudentSummary handles GET /students/:id/summary
func getStudentSummary(c *gin.Context) {
	idParam := c.Param("id")
	id, err := strconv.Atoi(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID"})
		return
	}

	mu.Lock()
	defer mu.Unlock()
	for _, student := range students {
		if student.ID == id {
			summary, err := generateSummary(student)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate summary"})
				return
			}
			c.JSON(http.StatusOK, gin.H{"summary": summary})
			return
		}
	}

	c.JSON(http.StatusNotFound, gin.H{"error": "Student not found"})
}

// generateSummary generates a summary of a student's profile using Ollama API
func generateSummary(student Student) (string, error) {
	prompt := fmt.Sprintf("Summarize the following student profile:\n\nID: %d\nName: %s\nAge: %d\nEmail: %s",
		student.ID, student.Name, student.Age, student.Email)

	requestBody, err := json.Marshal(map[string]string{
		"prompt": prompt,
		"model":  "llama2", // Replace with your actual model name
	})
	if err != nil {
		return "", err
	}

	resp, err := http.Post("http://localhost:11434/api/generate", "application/json", bytes.NewBuffer(requestBody))
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	responseBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	var response struct {
		Response string `json:"response"`
	}
	if err := json.Unmarshal(responseBody, &response); err != nil {
		return "", err
	}

	return response.Response, nil
}
