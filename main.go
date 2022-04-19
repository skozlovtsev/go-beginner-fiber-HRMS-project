package main

import (
	"context"
	"log"
	"time"

	"github.com/gofiber/fiber/v2"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type MongoInstance struct {
	Client *mongo.Client
	Db     *mongo.Database
}

var mg MongoInstance

const dbName = "fiber-hrms"
const mongoURI = "mongodb://localhost:27017/" + dbName

type Employee struct {
	ID     string  `json:"id, omitempty" bson:"_id, omitempty"`
	Name   string  `json:"name"`
	Salary float64 `json:"salary"`
	Age    float64 `json:"age"`
}

func Connect() error {
	client, err := mongo.NewClient(options.Client().ApplyURI(mongoURI))  //создаем новый плиент для подключения к базе данных mongoDB по  URI
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	err = client.Connect(ctx)
	db := client.Database(dbName)

	if err != nil {
		return err
	}

	mg = MongoInstance{  //создаем экземпляр MongoInstance
		Client: client,
		Db:     db,
	}
	return nil
}

func main() {
	if err := Connect(); err != nil {
		log.Fatal(err)
	}
	app := fiber.New()

	app.Get("/employee", func(c *fiber.Ctx) error {  //добавляем обработчик для метода GET

		query := bson.D{{}}  //создаем репрезентацию bson документа в качестве запроса к базе данных

		cursor, err := mg.Db.Collection("employees").Find(c.Context(), query)  //создаем курсор для взаимодействия с коллекцией employees
		if err != nil {
			return c.Status(500).SendString(err.Error())
		}

		var employees []Employee = make([]Employee, 0)  //создаем новый масив employees типа Employee

		if err := cursor.All(c.Context(), &employees); err != nil {
			return c.Status(500).SendString(err.Error())
		}

		return c.JSON(employees)  //отправляем employees в формате json в случае возникновения ошибки возвращаем ее
	})
	app.Post("/employee", func(c *fiber.Ctx) error {  //добавляем обработчик для метода POST
		collection := mg.Db.Collection("employees")  //получаем в переменную collection обработчик для коллекции employees

		employee := new(Employee)

		if err := c.BodyParser(employee); err != nil {
			return c.Status(400).SendString(err.Error())
		}

		employee.ID = ""

		insertionResult, err := collection.InsertOne(c.Context(), employee)  //добавляем в коллекцию employees новую запись

		if err != nil {
			return c.Status(500).SendString(err.Error())
		}

		filter := bson.D{{Key: "_id", Value: insertionResult.InsertedID}}  //создаем репрезентацию bson документа в качестве запроса фильтра
		createdRecord := collection.FindOne(c.Context(), filter)  //

		createdEmployee := &Employee{}
		createdRecord.Decode(createdEmployee)

		return c.Status(201).JSON(createdEmployee)  //отправляем createdEmployee в формате json с статусом 201 в случае возникновения ошибки возвращаем ее

	})
	app.Put("/employee/:id", func(c *fiber.Ctx) error {  //добавляем обработчик для метода PUT
		idParam := c.Params("id")  //получаем из передаваемого запроса параметр id

		employeeID, err := primitive.ObjectIDFromHex(idParam)  //генерируем новый id

		if err != nil {
			return c.SendStatus(400)
		}

		employee := new(Employee)

		if err := c.BodyParser(employee); err != nil {
			return c.Status(400).SendString(err.Error())
		}

		query := bson.D{{Key: "_id", Value: employeeID}}  //создаем репрезентацию bson документа в качестве запроса к базе данных
		update := bson.D{  //создаем репрезентацию bson документа в качестве новой записи
			{Key: "$set",
				Value: bson.D{
					{Key: "name", Value: employee.Name},
					{Key: "age", Value: employee.Age},
					{Key: "salary", Value: employee.Salary},
				},
			},
		}

		err = mg.Db.Collection("employees").FindOneAndUpdate(c.Context(), query, update).Err()  //заменяем соответствующий запросу query документ в коллекции employees

		if err != nil {
			if err == mongo.ErrNoDocuments {
				return c.SendStatus(400)
			}
			return c.SendStatus(500)
		}

		employee.ID = idParam

		return c.Status(200).JSON(employee)  //отправляем createdEmployee в формате json с статусом 200 в случае возникновения ошибки возвращаем ее

	})
	app.Delete("/employee/:id", func(c *fiber.Ctx) error {  //добавляем обработчик для метода DELETE

		employeeID, err := primitive.ObjectIDFromHex(c.Params("id"))  //генерируем новый id

		if err != nil {
			return c.SendStatus(400)
		}

		query := bson.D{{Key: "_id", Value: employeeID}}  //создаем репрезентацию bson документа в качестве запроса к базе данных
		result, err := mg.Db.Collection("employees").DeleteOne(c.Context(), &query)  //из коллекции employees удаляем элемент соответствующий запросу query

		if err != nil {
			return c.SendStatus(500)
		}

		if result.DeletedCount < 1 {
			return c.SendStatus(404)
		}

		return c.Status(200).JSON("record deleted")  //отправляем createdEmployee в формате json с статусом 200 в случае возникновения ошибки возвращаем ее

	})

	log.Fatal(app.Listen(":3000"))
}